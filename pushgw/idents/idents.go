package idents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ccfos/nightingale/v6/models"
	"github.com/ccfos/nightingale/v6/pkg/ctx"
	"github.com/ccfos/nightingale/v6/pkg/poster"
	"github.com/ccfos/nightingale/v6/storage"

	"github.com/toolkits/pkg/logger"
	"github.com/toolkits/pkg/slice"
)

type Set struct {
	sync.Mutex
	items map[string]struct{}
	redis storage.Redis
	ctx   *ctx.Context
}

func New(ctx *ctx.Context, redis storage.Redis) *Set {
	set := &Set{
		items: make(map[string]struct{}),
		redis: redis,
		ctx:   ctx,
	}

	set.Init()
	return set
}

func (s *Set) Init() {
	go s.LoopPersist()
}

func (s *Set) MSet(items map[string]struct{}) {
	s.Lock()
	defer s.Unlock()
	for ident := range items {
		s.items[ident] = struct{}{}
	}
}

func (s *Set) LoopPersist() {
	for {
		time.Sleep(time.Second)
		s.persist()
	}
}

func (s *Set) persist() {
	var items map[string]struct{}

	s.Lock()
	if len(s.items) == 0 {
		s.Unlock()
		return
	}

	items = s.items
	s.items = make(map[string]struct{})
	s.Unlock()

	s.updateTimestamp(items)
}

func (s *Set) updateTimestamp(items map[string]struct{}) {
	lst := make([]string, 0, 100)
	now := time.Now().Unix()
	num := 0
	for ident := range items {
		lst = append(lst, ident)
		num++
		if num == 100 {
			if err := s.UpdateTargets(lst, now); err != nil {
				logger.Errorf("failed to update targets: %v", err)
			}
			// 通过 lst[:0] 的方式重新切片，将切片的长度置为 0，但容量保持不变。
			// 这样做的目的可能是在重新利用切片时，清空其现有元素但保留分配的内存空间，
			// 以便后续可以更有效地添加新元素，而无需重新分配内存
			lst = lst[:0]
			num = 0
		}
	}

	if err := s.UpdateTargets(lst, now); err != nil {
		logger.Errorf("failed to update targets: %v", err)
	}
}

type TargetUpdate struct {
	Lst []string `json:"lst"`
	Now int64    `json:"now"`
}

func (s *Set) UpdateTargets(lst []string, now int64) error {
	err := updateTargetsUpdateTs(lst, now, s.redis)
	if err != nil {
		logger.Errorf("failed to update targets:%v update_ts: %v", lst, err)
	}

	if !s.ctx.IsCenter { // 边缘侧部署的n9e需要向中心服务进行上报
		t := TargetUpdate{
			Lst: lst,
			Now: now,
		}
		err := poster.PostByUrls(s.ctx, "/v1/n9e/target-update", t)
		return err
	}

	count := int64(len(lst))
	if count == 0 {
		return nil
	}
	// 更新target表中的数据（如果对应的监控对象的标识在表中）
	ret := s.ctx.DB.Table("target").Where("ident in ?", lst).Update("update_at", now)
	if ret.Error != nil {
		return ret.Error
	}

	if ret.RowsAffected == count {
		return nil
	}

	// there are some idents not found in db, so insert them
	var exists []string
	err = s.ctx.DB.Table("target").Where("ident in ?", lst).Pluck("ident", &exists).Error
	if err != nil {
		return err
	}

	news := slice.SubString(lst, exists)
	for i := 0; i < len(news); i++ {
		err = s.ctx.DB.Exec("INSERT INTO target(ident, update_at) VALUES(?, ?)", news[i], now).Error
		if err != nil {
			logger.Error("failed to insert target:", news[i], "error:", err)
		}
	}

	return nil
}

func updateTargetsUpdateTs(lst []string, now int64, redis storage.Redis) error {
	if redis == nil {
		return fmt.Errorf("redis is nil")
	}

	count := int64(len(lst))
	if count == 0 {
		return nil
	}

	newMap := make(map[string]interface{}, count)
	for _, ident := range lst {
		hostUpdateTime := models.HostUpdteTime{
			UpdateTime: now,
			Ident:      ident,
		}
		newMap[models.WrapIdentUpdateTime(ident)] = hostUpdateTime
	}

	err := storage.MSet(context.Background(), redis, newMap)
	return err
}
