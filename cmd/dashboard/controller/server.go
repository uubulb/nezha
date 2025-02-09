package controller

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"

	"github.com/nezhahq/nezha/model"
	"github.com/nezhahq/nezha/pkg/utils"
	pb "github.com/nezhahq/nezha/proto"
	"github.com/nezhahq/nezha/service/singleton"
)

// List server
// @Summary List server
// @Security BearerAuth
// @Schemes
// @Description List server
// @Tags auth required
// @Param id query uint false "Resource ID"
// @Produce json
// @Success 200 {object} model.CommonResponse[[]model.Server]
// @Router /server [get]
func listServer(c *gin.Context) ([]*model.Server, error) {
	slist := singleton.ServerShared.GetSortedList()

	var ssl []*model.Server
	if err := copier.Copy(&ssl, &slist); err != nil {
		return nil, err
	}
	return ssl, nil
}

// Edit server
// @Summary Edit server
// @Security BearerAuth
// @Schemes
// @Description Edit server
// @Tags auth required
// @Accept json
// @Param id path uint true "Server ID"
// @Param body body model.ServerForm true "ServerForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /server/{id} [patch]
func updateServer(c *gin.Context) (any, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return nil, err
	}
	var sf model.ServerForm
	if err := c.ShouldBindJSON(&sf); err != nil {
		return nil, err
	}

	singleton.DDNSCacheLock.RLock()
	for _, pid := range sf.DDNSProfiles {
		if p, ok := singleton.DDNSCache[pid]; ok {
			if !p.HasPermission(c) {
				singleton.DDNSCacheLock.RUnlock()
				return nil, singleton.Localizer.ErrorT("permission denied")
			}
		}
	}
	singleton.DDNSCacheLock.RUnlock()

	var s model.Server
	if err := singleton.DB.First(&s, id).Error; err != nil {
		return nil, singleton.Localizer.ErrorT("server id %d does not exist", id)
	}

	if !s.HasPermission(c) {
		return nil, singleton.Localizer.ErrorT("permission denied")
	}

	s.Name = sf.Name
	s.DisplayIndex = sf.DisplayIndex
	s.Note = sf.Note
	s.PublicNote = sf.PublicNote
	s.HideForGuest = sf.HideForGuest
	s.EnableDDNS = sf.EnableDDNS
	s.DDNSProfiles = sf.DDNSProfiles
	s.OverrideDDNSDomains = sf.OverrideDDNSDomains

	ddnsProfilesRaw, err := utils.Json.Marshal(s.DDNSProfiles)
	if err != nil {
		return nil, err
	}
	s.DDNSProfilesRaw = string(ddnsProfilesRaw)

	overrideDomainsRaw, err := utils.Json.Marshal(sf.OverrideDDNSDomains)
	if err != nil {
		return nil, err
	}
	s.OverrideDDNSDomainsRaw = string(overrideDomainsRaw)

	if err := singleton.DB.Save(&s).Error; err != nil {
		return nil, newGormError("%v", err)
	}

	s.CopyFromRunningServer(singleton.ServerShared.GetList()[s.ID])
	singleton.ServerShared.Update(&s, "")

	return nil, nil
}

// Batch delete server
// @Summary Batch delete server
// @Security BearerAuth
// @Schemes
// @Description Batch delete server
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[any]
// @Router /batch-delete/server [post]
func batchDeleteServer(c *gin.Context) (any, error) {
	var servers []uint64
	if err := c.ShouldBindJSON(&servers); err != nil {
		return nil, err
	}

	slist := singleton.ServerShared.GetList()
	for _, sid := range servers {
		if s, ok := slist[sid]; ok {
			if !s.HasPermission(c) {
				return nil, singleton.Localizer.ErrorT("permission denied")
			}
		}
	}

	err := singleton.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Delete(&model.Server{}, "id in (?)", servers).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&model.ServerGroupServer{}, "server_id in (?)", servers).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, newGormError("%v", err)
	}

	singleton.AlertsLock.Lock()
	for _, sid := range servers {
		for _, alert := range singleton.Alerts {
			if singleton.AlertsCycleTransferStatsStore[alert.ID] != nil {
				delete(singleton.AlertsCycleTransferStatsStore[alert.ID].ServerName, sid)
				delete(singleton.AlertsCycleTransferStatsStore[alert.ID].Transfer, sid)
				delete(singleton.AlertsCycleTransferStatsStore[alert.ID].NextUpdate, sid)
			}
		}
	}
	singleton.DB.Unscoped().Delete(&model.Transfer{}, "server_id in (?)", servers)
	singleton.AlertsLock.Unlock()

	singleton.ServerShared.Delete(servers)
	return nil, nil
}

// Force update Agent
// @Summary Force update Agent
// @Security BearerAuth
// @Schemes
// @Description Force update Agent
// @Tags auth required
// @Accept json
// @param request body []uint64 true "id list"
// @Produce json
// @Success 200 {object} model.CommonResponse[model.ServerTaskResponse]
// @Router /force-update/server [post]
func forceUpdateServer(c *gin.Context) (*model.ServerTaskResponse, error) {
	var forceUpdateServers []uint64
	if err := c.ShouldBindJSON(&forceUpdateServers); err != nil {
		return nil, err
	}

	forceUpdateResp := new(model.ServerTaskResponse)

	slist := singleton.ServerShared.GetList()
	for _, sid := range forceUpdateServers {
		server := slist[sid]
		if server != nil && server.TaskStream != nil {
			if !server.HasPermission(c) {
				return nil, singleton.Localizer.ErrorT("permission denied")
			}
			if err := server.TaskStream.Send(&pb.Task{
				Type: model.TaskTypeUpgrade,
			}); err != nil {
				forceUpdateResp.Failure = append(forceUpdateResp.Failure, sid)
			} else {
				forceUpdateResp.Success = append(forceUpdateResp.Success, sid)
			}
		} else {
			forceUpdateResp.Offline = append(forceUpdateResp.Offline, sid)
		}
	}

	return forceUpdateResp, nil
}

// Get server config
// @Summary Get server config
// @Security BearerAuth
// @Schemes
// @Description Get server config
// @Tags auth required
// @Produce json
// @Success 200 {object} model.CommonResponse[string]
// @Router /server/config/{id} [get]
func getServerConfig(c *gin.Context) (string, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return "", err
	}

	slist := singleton.ServerShared.GetList()
	s, ok := slist[id]
	if !ok || s.TaskStream == nil {
		return "", nil
	}

	if !s.HasPermission(c) {
		return "", singleton.Localizer.ErrorT("permission denied")
	}

	if err := s.TaskStream.Send(&pb.Task{
		Type: model.TaskTypeReportConfig,
	}); err != nil {
		return "", err
	}

	timeout := time.NewTimer(time.Second * 10)
	select {
	case <-timeout.C:
		return "", singleton.Localizer.ErrorT("operation timeout")
	case data := <-s.ConfigCache:
		timeout.Stop()
		switch data := data.(type) {
		case string:
			return data, nil
		case error:
			return "", singleton.Localizer.ErrorT("get server config failed: %v", data)
		}
	}

	return "", singleton.Localizer.ErrorT("get server config failed")
}

// Set server config
// @Summary Set server config
// @Security BearerAuth
// @Schemes
// @Description Set server config
// @Tags auth required
// @Accept json
// @Param body body model.ServerConfigForm true "ServerConfigForm"
// @Produce json
// @Success 200 {object} model.CommonResponse[model.ServerTaskResponse]
// @Router /server/config [post]
func setServerConfig(c *gin.Context) (*model.ServerTaskResponse, error) {
	var configForm model.ServerConfigForm
	if err := c.ShouldBindJSON(&configForm); err != nil {
		return nil, err
	}

	var resp model.ServerTaskResponse
	slist := singleton.ServerShared.GetList()
	servers := make([]*model.Server, 0, len(configForm.Servers))
	for _, sid := range configForm.Servers {
		if s, ok := slist[sid]; ok {
			if !s.HasPermission(c) {
				return nil, singleton.Localizer.ErrorT("permission denied")
			}
			if s.TaskStream == nil {
				resp.Offline = append(resp.Offline, s.ID)
				continue
			}
			servers = append(servers, s)
		}
	}

	var wg sync.WaitGroup
	var respMu sync.Mutex

	for i := 0; i < len(servers); i += 10 {
		end := i + 10
		if end > len(servers) {
			end = len(servers)
		}
		group := servers[i:end]

		wg.Add(1)
		go func(srvGroup []*model.Server) {
			defer wg.Done()
			for _, s := range srvGroup {
				// Create and send the task.
				task := &pb.Task{
					Type: model.TaskTypeApplyConfig,
					Data: configForm.Config,
				}
				if err := s.TaskStream.Send(task); err != nil {
					respMu.Lock()
					resp.Failure = append(resp.Failure, s.ID)
					respMu.Unlock()
					continue
				}
				respMu.Lock()
				resp.Success = append(resp.Success, s.ID)
				respMu.Unlock()
			}
		}(group)
	}

	wg.Wait()
	return &resp, nil
}
