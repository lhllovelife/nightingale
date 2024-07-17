package metas

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/storage"

	"github.com/toolkits/pkg/logger"
)

type Set struct {
	sync.RWMutex
	items map[string]models.HostMeta
	redis storage.Redis
}

func New(redis storage.Redis) *Set {
	set := &Set{
		items: make(map[string]models.HostMeta),
		redis: redis,
	}

	set.Init()
	return set
}

func (s *Set) Init() {
	go s.LoopPersist()
}

func (s *Set) MSet(items map[string]models.HostMeta) {
	s.Lock()
	defer s.Unlock()
	for ident, meta := range items {
		s.items[ident] = meta
	}
}

func (s *Set) Set(ident string, meta models.HostMeta) {
	s.Lock()
	defer s.Unlock()
	s.items[ident] = meta
}

// 持续将数据持久化到 Redis中, ，每秒调用一次persist()方法
func (s *Set) LoopPersist() {
	for {
		time.Sleep(time.Second)
		s.persist()
	}
}

// 将该实例内存中存储的所有主机元数据，进行持久化
func (s *Set) persist() {
	var items map[string]models.HostMeta

	s.Lock()
	if len(s.items) == 0 { // 无数据，返回
		s.Unlock()
		return
	}

	items = s.items
	s.items = make(map[string]models.HostMeta)
	s.Unlock()

	s.updateMeta(items)
}

// 将 HostMeta 对象按 100 个为一组批量更新到 Redis 中，不足100的，则按照一个批次持久化。
// 如果有扩展信息（ExtendInfo），则单独处理并存储
func (s *Set) updateMeta(items map[string]models.HostMeta) {
	m := make(map[string]models.HostMeta, 100) // 定义一个初使容量为100个的map
	num := 0

	for _, meta := range items {
		m[meta.Hostname] = meta
		num++
		if num == 100 {
			if err := s.updateTargets(m); err != nil {
				logger.Errorf("failed to update targets: %v", err)
			}
			m = make(map[string]models.HostMeta, 100)
			num = 0
		}
	}

	if err := s.updateTargets(m); err != nil {
		logger.Errorf("failed to update targets: %v", err)
	}
}

// updateTargets 方法将处理后的 HostMeta 数据使用 storage.MSet 方法批量写入 Redis。如果 ExtendInfo 存在，也会单独写入 Redis。
// 参数key: n9e_meta_{hostname} 参数value: 去除了扩展信息的主机信息
// 参数key: n9e_extend_meta_{hostname} 参数value: 扩展信息
func (s *Set) updateTargets(m map[string]models.HostMeta) error {
	if s.redis == nil {
		logger.Warningf("redis is nil")
		return nil
	}

	count := int64(len(m))
	if count == 0 {
		return nil
	}

	newMap := make(map[string]interface{}, count)
	extendMap := make(map[string]interface{})
	for ident, meta := range m {
		if meta.ExtendInfo != nil {
			extendMeta := meta.ExtendInfo
			meta.ExtendInfo = make(map[string]interface{})
			extendMetaStr, err := json.Marshal(extendMeta)
			if err != nil {
				return err
			}
			extendMap[models.WrapExtendIdent(ident)] = extendMetaStr
		}
		newMap[models.WrapIdent(ident)] = meta
	}
	err := storage.MSet(context.Background(), s.redis, newMap) // 通过 go-redis Pipeline 一次执行多个命令
	if err != nil {
		return err
	}

	if len(extendMap) > 0 {
		err = storage.MSet(context.Background(), s.redis, extendMap)
		if err != nil {
			return err
		}
	}

	return err
}
