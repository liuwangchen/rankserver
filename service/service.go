package service

import (
	"context"

	"github.com/liuwangchen/apis/apipb"
	"github.com/liuwangchen/rankserver/config"
	"github.com/liuwangchen/rankserver/logic"
)

type RankService struct {
}

func NewRankService() *RankService {
	return &RankService{}
}

// 检查rankType合法性
func (this *RankService) checkRankType(rankType int32) bool {
	rankTypeRange := config.GetInstance().Rank.Dynamic.TypeRange
	if rankType >= rankTypeRange[0] && rankType <= rankTypeRange[1] {
		return true
	}
	return false
}

func (this *RankService) GetRank(ctx context.Context, req *apipb.ReqGetRank) (*apipb.RspGetRank, error) {
	if !this.checkRankType(req.RankType) {
		return &apipb.RspGetRank{Code: apipb.RET_RankTypeErr}, nil
	}
	if req.BeginRank <= 0 {
		return &apipb.RspGetRank{Code: apipb.RET_RankBeginInputErr}, nil
	}
	return logic.GetRankManagerInstance().GetRank(req.RankType, int(req.BeginRank), int(req.Count), req.Me, req.Reverse), nil
}

func (this *RankService) UpdateRank(ctx context.Context, req *apipb.ReqUpdateRank) (*apipb.CommonRsp, error) {
	if !this.checkRankType(req.RankType) {
		return &apipb.CommonRsp{Code: apipb.RET_RankTypeErr}, nil
	}
	logic.GetRankManagerInstance().UpdateRankScore(req.RankType, req.RankData)
	return &apipb.CommonRsp{}, nil
}

func (this *RankService) DeleteRankMems(ctx context.Context, req *apipb.ReqDeleteRankMems) (*apipb.CommonRsp, error) {
	if !this.checkRankType(req.RankType) {
		return &apipb.CommonRsp{Code: apipb.RET_RankTypeErr}, nil
	}
	logic.GetRankManagerInstance().DeleteRankData(req.RankType, req.Mems...)
	return &apipb.CommonRsp{}, nil
}

func (this *RankService) GetRankByOffset(ctx context.Context, req *apipb.ReqGetRankByOffset) (*apipb.RspGetRank, error) {
	if !this.checkRankType(req.RankType) {
		return &apipb.RspGetRank{Code: apipb.RET_RankTypeErr}, nil
	}
	return logic.GetRankManagerInstance().GetRankByOffset(req.RankType, req.Me, int(req.Offset), req.Reverse), nil
}

func (this *RankService) DeleteRank(ctx context.Context, req *apipb.ReqDeleteRankMems) (*apipb.CommonRsp, error) {
	if !this.checkRankType(req.RankType) {
		return &apipb.CommonRsp{Code: apipb.RET_RankTypeErr}, nil
	}
	code := logic.GetRankManagerInstance().DeleteRank(req.RankType)
	return &apipb.CommonRsp{Code: code}, nil
}
