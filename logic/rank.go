package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/liuwangchen/apis/apipb"
	"github.com/liuwangchen/rankserver/components"
	"github.com/liuwangchen/rankserver/config"
	"github.com/liuwangchen/rankserver/pb"
	"github.com/liuwangchen/toy/pkg/container/zset"
	"github.com/liuwangchen/toy/third_party/redisx"
)

var (
	instance *RankManager
	once     sync.Once
)

func GetRankManagerInstance() *RankManager {
	once.Do(func() {
		instance = newRankManager()
	})

	return instance
}

type RankManager struct {
	rankTypes []int32
	ranks     map[string]*zset.ZSet
}

func newRankManager() *RankManager {
	return &RankManager{ranks: make(map[string]*zset.ZSet)}
}

func (this *RankManager) Init(ctx context.Context) error {
	err := this.load(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (this *RankManager) getSaveRankTypeKey() string {
	return fmt.Sprintf("%s_rankType", config.GetInstance().Rank.Dynamic.ServerId)
}

// 加载排行榜数据
func (this *RankManager) load(ctx context.Context) error {
	// 加载所有排行榜类型
	b, err := components.RedisClient.Sync().Get(this.getSaveRankTypeKey()).Bytes()
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	err = json.Unmarshal(b, &this.rankTypes)
	if err != nil {
		return err
	}

	// 遍历所有排行榜key，load排行榜数据到zset
	for _, rankType := range this.rankTypes {
		rankKey := this.getRankKey(rankType)
		rankData := components.RedisClient.Sync().HGetAll(rankKey)
		if rankData.Err() != nil {
			return rankData.Err()
		}
		z, ok := this.ranks[rankKey]
		if !ok {
			z = zset.New()
			this.ranks[rankKey] = z
		}
		err := rankData.ForEachStringBytes(func(mem string, b []byte) error {
			data := new(pb.DBRankItem)
			err := proto.Unmarshal(b, data)
			if err != nil {
				return err
			}
			_, err = z.Add(mem, data)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *RankManager) getRankKey(rankType int32) string {
	return fmt.Sprintf("rank_%d", rankType)
}

// 获取或创建排行榜
func (this *RankManager) getOrCreateRank(rankType int32) *zset.ZSet {
	rankKey := this.getRankKey(rankType)
	z, ok := this.ranks[rankKey]
	if !ok {
		z = zset.New()
		this.ranks[rankKey] = z

		// 更新
		this.rankTypes = append(this.rankTypes, rankType)
		b, err := json.Marshal(this.rankTypes)
		if err != nil {
			return nil
		}
		_ = components.RedisClient.Set(context.Background(), this.getSaveRankTypeKey(), b)
	}
	return z
}

// UpdateRankScore 更新排行榜，替换形势的
func (this *RankManager) UpdateRankScore(rankType int32, rankData map[string]*apipb.RankChangeData) {
	rankKey := this.getRankKey(rankType)
	// 获取zset
	z := this.getOrCreateRank(rankType)
	savedata := make(map[string]string, len(rankData))
	for mem, v := range rankData {
		rankItem := new(pb.DBRankItem)
		// 更新zset
		rankItem.Key = mem
		rankItem.Score = uint64(v.Score)
		if v.Then > 0 {
			rankItem.Then = uint64(v.Then)
		}
		if _, err := z.Add(mem, rankItem); err != nil {
			continue
		}

		// 保存数据
		b, err := proto.Marshal(rankItem)
		if err != nil {
			continue
		}
		savedata[mem] = string(b)
	}
	components.RedisClient.HMSet(context.Background(), rankKey, redisx.NewMapReqStringString(savedata))
}

// DeleteRankData 删除排行榜数据
func (this *RankManager) DeleteRankData(rankType int32, mems ...string) {
	rankKey := this.getRankKey(rankType)
	z := this.ranks[rankKey]
	if z == nil {
		return
	}

	// 删除zset
	for _, mem := range mems {
		if _, err := z.Remove(mem); err != nil {
			continue
		}
	}

	// 删除db
	components.RedisClient.HDel(context.Background(), rankKey, mems...)
}

func (this *RankManager) toRankItem(item *pb.DBRankItem, rank int) *apipb.RankItem {
	return &apipb.RankItem{
		Id:    item.Key,
		Score: int64(item.Score),
		Rank:  uint32(rank), // item的rank从0开始的
		Then:  int64(item.Then),
	}
}

func (this *RankManager) getRank(rankType int32, startIndex int, endIndex int, me string, reverse bool) *apipb.RspGetRank {
	rankKey := this.getRankKey(rankType)
	result := new(apipb.RspGetRank)
	z := this.ranks[rankKey]
	if z == nil {
		return result
	}

	// 范围查询，[start,end]
	z.Range(startIndex, endIndex, reverse, func(key string, i zset.Item, rank int) bool {
		item := i.(*pb.DBRankItem)
		result.Ranks = append(result.Ranks, this.toRankItem(item, rank))
		return true
	})

	// top1查询
	z.Range(0, 0, reverse, func(key string, i zset.Item, rank int) bool {
		item := i.(*pb.DBRankItem)
		result.Top = this.toRankItem(item, rank)
		return true
	})

	// me查询
	meItem := z.Get(me)
	if meItem != nil {
		item := meItem.(*pb.DBRankItem)
		// 自己排行
		meRank := z.Rank(me, reverse)
		result.Me = this.toRankItem(item, meRank)
	}

	// 排行榜总数量
	result.TotalRankNum = uint32(z.Length())
	return result
}

// GetRank 获取排行榜
func (this *RankManager) GetRank(rankType int32, startRank int, count int, me string, reverse bool) *apipb.RspGetRank {
	rankKey := this.getRankKey(rankType)
	result := new(apipb.RspGetRank)
	z := this.ranks[rankKey]
	if z == nil {
		return result
	}
	startIndex := startRank - 1
	endIndex := startIndex + count - 1

	return this.getRank(rankType, startIndex, endIndex, me, reverse)
}

// GetRankByOffset 获取me的前后offset名次的排行榜
func (this *RankManager) GetRankByOffset(rankType int32, me string, offset int, reverse bool) *apipb.RspGetRank {
	rankKey := this.getRankKey(rankType)
	result := new(apipb.RspGetRank)
	z := this.ranks[rankKey]
	if z == nil {
		return result
	}
	// me查询
	meItem := z.Get(me)
	if meItem == nil {
		return result
	}
	// 找到自己排行
	meRank := z.Rank(me, reverse)

	// 偏移
	startIndex := meRank - offset - 1
	if startIndex < 0 {
		startIndex = 0
	}
	endIndex := meRank + offset - 1

	return this.getRank(rankType, startIndex, endIndex, me, reverse)
}

// DeleteRank 删除排行榜
func (this *RankManager) DeleteRank(rankType int32) apipb.RET {
	rankKey := this.getRankKey(rankType)
	z := this.ranks[rankKey]
	if z == nil {
		return apipb.RET_ERROR
	}
	delete(this.ranks, rankKey)
	components.RedisClient.Del(context.Background(), rankKey)
	return apipb.RET_OK
}
