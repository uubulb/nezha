package singleton

import (
	"cmp"
	"maps"
	"slices"
	"sync"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/pkg/utils"
)

type ServerClass struct {
	list     map[uint64]*model.Server
	uuidToID map[string]uint64
	listMu   sync.RWMutex

	sortedList         []*model.Server
	sortedListForGuest []*model.Server
	sortedListMu       sync.RWMutex
}

func NewServerClass() *ServerClass {
	sc := &ServerClass{
		list:     make(map[uint64]*model.Server),
		uuidToID: make(map[string]uint64),
	}

	var servers []model.Server
	DB.Find(&servers)
	for _, s := range servers {
		innerS := s
		model.InitServer(&innerS)
		sc.list[innerS.ID] = &innerS
		sc.uuidToID[innerS.UUID] = innerS.ID
	}
	sc.sortList()

	return sc
}

func (c *ServerClass) Update(s *model.Server, uuid string) {
	c.listMu.Lock()

	c.list[s.ID] = s
	if uuid != "" {
		c.uuidToID[uuid] = s.ID
	}

	c.listMu.Unlock()

	c.sortList()
}

func (c *ServerClass) Delete(idList []uint64) {
	c.listMu.Lock()

	for _, id := range idList {
		serverUUID := c.list[id].UUID
		delete(c.uuidToID, serverUUID)
		delete(c.list, id)
	}

	c.listMu.Unlock()

	c.sortList()
}

func (c *ServerClass) GetServer(id uint64) (s *model.Server, ok bool) {
	c.listMu.RLock()
	defer c.listMu.RUnlock()

	s, ok = c.list[id]
	return
}

func (c *ServerClass) GetList() map[uint64]*model.Server {
	c.listMu.RLock()
	defer c.listMu.RUnlock()

	return maps.Clone(c.list)
}

func (c *ServerClass) GetSortedList() []*model.Server {
	c.sortedListMu.RLock()
	defer c.sortedListMu.RUnlock()

	return slices.Clone(c.sortedList)
}

func (c *ServerClass) GetSortedListForGuest() []*model.Server {
	c.sortedListMu.RLock()
	defer c.sortedListMu.RUnlock()

	return slices.Clone(c.sortedListForGuest)
}

func (c *ServerClass) UUIDToID(uuid string) (id uint64, ok bool) {
	c.listMu.RLock()
	defer c.listMu.RUnlock()

	id, ok = c.uuidToID[uuid]
	return
}

func (c *ServerClass) sortList() {
	c.listMu.RLock()
	defer c.listMu.RUnlock()
	c.sortedListMu.Lock()
	defer c.sortedListMu.Unlock()

	c.sortedList = utils.MapValuesToSlice(c.list)
	// 按照服务器 ID 排序的具体实现（ID越大越靠前）
	slices.SortStableFunc(c.sortedList, func(a, b *model.Server) int {
		if a.DisplayIndex == b.DisplayIndex {
			return cmp.Compare(a.ID, b.ID)
		}
		return cmp.Compare(b.DisplayIndex, a.DisplayIndex)
	})

	c.sortedListForGuest = make([]*model.Server, 0, len(c.sortedList))
	for _, s := range c.sortedList {
		if !s.HideForGuest {
			c.sortedListForGuest = append(c.sortedListForGuest, s)
		}
	}
}
