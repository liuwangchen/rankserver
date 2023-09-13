package pb

import (
	"github.com/liuwangchen/toy/pkg/container/zset"
)

func (item1 *DBRankItem) Less(item zset.Item) bool {
	item2 := item.(*DBRankItem)
	if item1.GetScore() == item2.GetScore() {
		if item1.GetThen() == item2.GetThen() {
			return item1.GetKey() < item2.GetKey()
		}
		return item1.GetThen() < item2.GetThen()
	}
	return item1.GetScore() < item2.GetScore()
}
