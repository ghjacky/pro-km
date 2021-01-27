package configserver

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/apis"

	"gorm.io/gorm"

	model2 "code.xxxxx.cn/platform/galaxy/pkg/manager/model"
	"github.com/kylelemons/godebug/diff"
	"github.com/magiconair/properties"

	"code.xxxxx.cn/platform/galaxy/pkg/manager/db"

	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/docker"
	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"

	"code.xxxxx.cn/platform/galaxy/pkg/component/cmd"
	"code.xxxxx.cn/platform/galaxy/pkg/component/storage/filedb"
	model "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"
)

/* All Import from Image Const */
const (
	tmpDir        = "/tmp/data"
	cameraInfoDir = "/root/CameraInfos"
	storeInfoDir  = "/root/StoreInfos"
)

type configManager struct {
	cmdServer cmd.Server
	storage   *filedb.FileDB
}

var formatMap = map[string]model.ConfigFormat{
	".properties": model.ConfigFormatProperties,
	".text":       model.ConfigFormatTxt,
	".txt":        model.ConfigFormatTxt,
	".yaml":       model.ConfigFormatYAML,
	".yml":        model.ConfigFormatYAML,
	".json":       model.ConfigFormatJSON,
	".csv":        model.ConfigFormatCSV,
	".png":        model.ConfigFormatJPG,
	".jpeg":       model.ConfigFormatJPG,
	".jpg":        model.ConfigFormatJPG,
}

// NewConfigManager create a new config manager
func NewConfigManager(dataDir string, cmdServer cmd.Server, stopCh <-chan struct{}) (Manager, *filedb.FileDB) {
	m := &configManager{
		cmdServer: cmdServer,
	}
	m.addCmdHandlers()
	m.newFsDb(dataDir)
	return m, m.storage
}

// FileMetaInfo .
type FileMetaInfo struct {
	Digest string            `json:"digest" yaml:"digest"`
	Name   string            `json:"name" yaml:"name"`
	Extend map[string]string `json:"extends" yaml:"extends"`
}

func (cm *configManager) newFsDb(dir string) {

	fileDB, err := filedb.NewFileDB(&filedb.Config{Workdir: dir})
	if err != nil {
		alog.Fatalf("create file db err: %v", err)
		return
	}
	cm.storage = fileDB
}

// AddConfig add new config by uploading or writing on page
func (cm *configManager) AddConfig(config *model.ConfigInfo) error {
	if valid, err := cm.checkConfigValidity(config); !valid {
		return err
	}

	// begin a transaction
	tx := db.NewTransaction()
	// 1. create or find ns
	if _, err := model.FindOrCreateConfigNamespace(config.Namespace, config.SiteID, config.CreatedBy, tx); err != nil {
		alog.Errorf("AddConfig find or create ns err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. create a config info
	config.Status = model.ConfigStatusCreated
	configInfo, err := model.CreateConfigInfo(config, tx)
	if err != nil {
		alog.Errorf("AddConfig create configInfo err: %v", err)
		tx.Rollback()
		return err
	}
	renderConfig, err := cm.renderClusterEnv(*config)
	if err != nil {
		alog.Errorf("AddConfig render configInfo err: %v", err)
		tx.Rollback()
		return err
	}
	// 3. put data to file db
	digest, replaceDigest, err := cm.storeConfigData(renderConfig, 1)
	if err != nil {
		alog.Errorf("AddConfig store data err: %v", err)
		tx.Rollback()
		return err
	}
	// 4. create a config version
	configVersion := &model.ConfigVersion{
		ConfigInfoID: configInfo.ID,
		SiteID:       renderConfig.SiteID,
		Name:         renderConfig.Name,
		Namespace:    renderConfig.Namespace,
		Version:      1,
		Desc:         renderConfig.Desc,
		CreatedBy:    renderConfig.CreatedBy,
		Digest:       digest,
		Status:       model.ConfigVersionCreate,
	}
	if renderConfig.TemplateName != "" && renderConfig.TemplateNamespace != "" {
		templateVersion, err := cm.getConfigTemplate(renderConfig)
		if err != nil {
			alog.Errorf("AddConfig get config template err: %v", err)
			tx.Rollback()
			return err
		}
		configVersion.TemplateVersionID = templateVersion.ID
		configVersion.TemplateNamespace = renderConfig.TemplateNamespace
		configVersion.TemplateName = renderConfig.TemplateName
		configVersion.ReplaceDigest = replaceDigest
	}
	if _, err := model.CreateConfigVersion(configVersion, tx); err != nil {
		alog.Errorf("AddConfig create config version err: %v", err)
		tx.Rollback()
		return err
	}
	// 5. create config current table
	configCurrent := &model.ConfigCurrent{
		ConfigInfoID:          configInfo.ID,
		LatestConfigVersionID: configVersion.ID,
		LatestReleaseID:       0,
		PreReleaseID:          0,
		Status:                "created",
	}
	if _, err := model.CreateConfigCurrent(configCurrent, tx); err != nil {
		alog.Errorf("AddConfig create config current err: %v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// CountConfigNamespaces return the number of config namespace
func (cm *configManager) CountConfigNamespaces(query string) (int64, error) {
	return model.CountConfigNamespace(query)
}

// ListConfigNamespaces return the list of config namespace
func (cm *configManager) ListConfigNamespaces(query string, orders []string, offset, limit int) ([]*model.ConfigNamespace, error) {
	configNs, err := model.ListConfigNamespace(query, orders, offset, limit)
	if err != nil {
		return nil, err
	}

	return configNs, nil
}

// CountConfigs return the number of config
func (cm *configManager) CountConfigs(query string) (int64, error) {

	return model.CountConfigInfos(query)
}

// ListConfigs return the list of config
func (cm *configManager) ListConfigs(query string, orders []string, offset, limit int) ([]*model.ConfigInfo, error) {

	configs, err := model.ListConfigInfo(query, orders, offset, limit)
	if err != nil {
		return nil, err
	}

	for _, config := range configs {

		if err := cm.checkIsTemplateChange(config); err != nil {
			alog.Errorf("ListConfigs check template change err: %v", err)
			continue
		}
	}

	return configs, nil
}

// UpdateConfig update config
func (cm *configManager) UpdateConfig(config *model.ConfigInfo) error {
	if valid, err := cm.checkConfigValidity(config); !valid {
		return err
	}
	renderConfig, err := cm.renderClusterEnv(*config)
	if err != nil {
		alog.Errorf("UpdateConfig render configInfo err: %v", err)
		return err
	}
	// begin a transaction
	tx := db.NewTransaction()
	// 1. get old version
	configCurrent, err := model.GetConfigCurrentByConfigID(config.ID)
	if err != nil {
		alog.Errorf("UpdateConfig get config current err: %v", err)
		tx.Rollback()
		return err
	}
	oldConfigVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
	if err != nil {
		alog.Errorf("UpdateConfig get old config version err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. put data to file db
	digest, replaceDigest, err := cm.storeConfigData(renderConfig, oldConfigVersion.Version+1)
	if err != nil {
		alog.Errorf("UpdateConfig store data err: %v", err)
		tx.Rollback()
		return err
	}
	// 3. create new version
	if cm.isConfigNeedNewVersion(oldConfigVersion, renderConfig, digest, replaceDigest) {
		newConfigVersion := &model.ConfigVersion{
			ConfigInfoID: renderConfig.ID,
			SiteID:       renderConfig.SiteID,
			Name:         renderConfig.Name,
			Namespace:    renderConfig.Namespace,
			Version:      oldConfigVersion.Version + 1,
			Desc:         renderConfig.Desc,
			CreatedBy:    renderConfig.CreatedBy,
			Digest:       digest,
			Status:       model.ConfigVersionUpdate,
		}
		if renderConfig.Format == model.ConfigFormatProperties &&
			renderConfig.TemplateVersionID != 0 &&
			renderConfig.TemplateName != "" && renderConfig.TemplateNamespace != "" {

			templateVersion, err := model.GetTemplateVersionByID(config.TemplateVersionID)
			if err != nil {
				alog.Errorf("UpdateConfig get config template version err: %v", err)
				tx.Rollback()
				return err
			}

			newConfigVersion.TemplateVersionID = templateVersion.ID
			newConfigVersion.TemplateNamespace = templateVersion.Namespace
			newConfigVersion.TemplateName = templateVersion.Name
			newConfigVersion.ReplaceDigest = replaceDigest

			config.TemplateVersionID = templateVersion.ID

		}
		if _, err := model.CreateConfigVersion(newConfigVersion, tx); err != nil {
			alog.Errorf("UpdateConfig create new config version err: %v", err)
			tx.Rollback()
			return err
		}

		configCurrent.LatestConfigVersionID = newConfigVersion.ID
		if config.Status != model.ConfigStatusCreated {
			config.Status = model.ConfigStatusUpdated
		}
	}
	// 4. update config info
	if err := model.UpdateConfigInfo(config, tx); err != nil {
		alog.Errorf("UpdateConfig update config info err: %v", err)
		tx.Rollback()
		return err
	}
	// 5. update config current table
	if err := model.UpdateConfigCurrent(configCurrent, tx); err != nil {
		alog.Errorf("UpdateConfig update config current err: %v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// ReleaseConfig release a config
func (cm *configManager) ReleaseConfig(id uint64, desc, creator string) error {

	// begin transaction
	tx := db.NewTransaction()
	// 1. get current status
	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("ReleaseConfig: get config info err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. release config
	release, err := cm.releaseConfig(configInfo, tx)
	if err != nil {
		alog.Errorf("ReleaseConfig: relese config err: %v", err)
		tx.Rollback()
		return err
	}

	configVersion, err := model.GetConfigVersionByID(release.ConfigVersionID)
	if err != nil {
		alog.Errorf("ReleaseConfig: get config version err: %v", err)
		tx.Rollback()
		return err
	}

	// 4. notify agent
	args := cmd.Args{
		"namespace": configVersion.Namespace,
		"filename":  configVersion.Name,
		"version":   fmt.Sprintf("%v", configVersion.Version),
	}
	executor, err := cm.getConnKeyFromSiteID(configInfo.SiteID)
	if err != nil {
		alog.Errorf("ReleaseConfig: get executor by site id err: %v ", err)
		tx.Rollback()
		return err
	}
	c := cm.cmdServer.NewCmdReq(cmd.CmdNSFileHandler, args, executor)
	resp, err := cm.cmdServer.SendSync(c, 3)
	if err != nil {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.CmdNSFileHandler, err)
		tx.Rollback()
		return err
	}
	if resp.Code == apis.FailCode {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.CmdNSFileHandler, resp.Msg)
		tx.Rollback()
		return fmt.Errorf(resp.Msg)
	}

	tx.Commit()
	return nil
}

// ReleaseConfigDetail get release config detail
func (cm *configManager) ReleaseConfigDetail(id uint64) (string, []*model.ConfigInstance, error) {

	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("ReleaseConfigDetail: get config info err: %v", err)
		return "", nil, err
	}

	return cm.getReleaseConfigDetail(configInfo)
}

// ReleaseNamespace release configs of namespace
func (cm *configManager) ReleaseNamespace(siteID string, ns string, desc, creator string) error {

	query := fmt.Sprintf("site_id=%q AND namespace=%q AND status IN ('created', 'updated')", siteID, ns)

	// 1. get configs whose status in created or updated
	configInfos, err := model.GetAllConfigInfos(query)
	if err != nil {
		alog.Errorf("ReleaseNamespace: get config infos err: %v", err)
		return err
	}

	// 2. create template tar
	distDir := fmt.Sprintf("%v", time.Now().Unix())
	if err := os.MkdirAll(distDir, os.ModePerm); err != nil {
		alog.Errorf("ReleaseNamespace: mkdir temp dir err: %v", err)
		return err
	}
	defer os.RemoveAll(distDir)

	distFile := filepath.Join(distDir, ns+".tar.gz")

	tx := db.NewTransaction()

	if err := cm.packConfigs(configInfos, distFile, tx); err != nil {
		alog.Errorf("ReleaseNamespace: pack configs err: %v", err)
		tx.Rollback()
		return err
	}

	// 3. store package to file db
	data, err := os.Open(distFile)
	if err != nil {
		alog.Errorf("ReleaseNamespace: open package err: %v", err)
		tx.Rollback()
		return err
	}
	digest, err := cm.storage.StorePackage(siteID, ns, data)
	if err != nil {
		alog.Errorf("ReleaseNamespace: store package err: %v", err)
		tx.Rollback()
		return err
	}
	// 4. notify agent
	args := cmd.Args{
		"namespace": ns,
		"digest":    digest,
	}
	executor, err := cm.getConnKeyFromSiteID(siteID)
	if err != nil {
		alog.Errorf("ReleaseNamespace: get executor by site id err: %v ", err)
		tx.Rollback()
		return err
	}
	c := cm.cmdServer.NewCmdReq(cmd.CmdNSPackageHandler, args, executor)
	resp, err := cm.cmdServer.SendSync(c, 3)
	if err != nil {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.CmdNSPackageHandler, err)
		tx.Rollback()
		return err
	}
	if resp.Code == apis.FailCode {
		alog.Errorf("Send Cmd %s failed: %v ", cmd.CmdNSPackageHandler, resp.Msg)
		tx.Rollback()
		return fmt.Errorf(resp.Msg)
	}

	tx.Commit()
	return nil
}

// ReleaseNamespaceDetail get release ns detail
func (cm *configManager) ReleaseNamespaceDetail(siteID string, ns string) ([]*model.ConfigAndInstance, error) {
	query := fmt.Sprintf("site_id=%q AND namespace=%q AND status IN ('created', 'updated')", siteID, ns)

	configInfos, err := model.GetAllConfigInfos(query)
	if err != nil {
		alog.Errorf("ReleaseNamespaceDetail: get config infos err: %v", err)
		return nil, err
	}

	var configAndInstances []*model.ConfigAndInstance
	for _, configInfo := range configInfos {
		instances, err := cm.getConfigInstances(configInfo)
		if err != nil {
			alog.Errorf("ReleaseNamespaceDetail: get config release detail err: %v", err)
			return nil, err
		}
		ci := &model.ConfigAndInstance{
			Config:    configInfo,
			Instances: instances,
		}
		configAndInstances = append(configAndInstances, ci)
	}

	return configAndInstances, nil
}

// RollbackConfig rollback a config
func (cm *configManager) RollbackConfig(id uint64, versionID uint64, creator string) error {

	if versionID != 0 {
		// TODO: support rollback to the specified version
		err := fmt.Errorf("rollback to the specified version is not currently supported")
		alog.Errorf("RollbackConfig: %v", err)
		return err
	}
	// begin transaction
	tx := db.NewTransaction()
	// 1. get current status
	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("RollbackConfig: get config info err: %v", err)
		tx.Rollback()
		return err
	}
	configCurrent, err := model.GetConfigCurrentByConfigID(id)
	if err != nil {
		alog.Errorf("RollbackConfig: get config current err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. get pre release
	if configCurrent.PreReleaseID == 0 {
		err := fmt.Errorf("there are no more rollback versions")
		alog.Errorf("RollbackConfig: there is no rollback version")
		tx.Rollback()
		return err
	}
	preRelease, err := model.GetReleaseByID(configCurrent.PreReleaseID)
	if err != nil {
		alog.Errorf("RollbackConfig: get pre release err: %v", err)
		tx.Rollback()
		return err
	}
	preConfigVersion, err := model.GetConfigVersionByID(preRelease.ConfigVersionID)
	if err != nil {
		alog.Errorf("RollbackConfig: get pre config version err: %v", err)
		tx.Rollback()
		return err
	}
	data, _, replaceData, err := cm.visitConfigVersionData(preConfigVersion)
	if err != nil {
		alog.Errorf("RollbackConfig: get pre config data err: %v", err)
		tx.Rollback()
		return err
	}
	latestConfigVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
	if err != nil {
		alog.Errorf("RollbackConfig: get latest config version err: %v", err)
		tx.Rollback()
		return err
	}
	// 3. create new config version
	newConfigVersion := &model.ConfigVersion{
		Version:           latestConfigVersion.Version + 1,
		Name:              preConfigVersion.Name,
		Namespace:         preConfigVersion.Namespace,
		SiteID:            preConfigVersion.SiteID,
		Desc:              preConfigVersion.Desc,
		CreatedBy:         creator,
		Group:             preConfigVersion.Group,
		Digest:            preConfigVersion.Digest,
		Status:            model.ConfigVersionRollback,
		ConfigInfoID:      preConfigVersion.ConfigInfoID,
		TemplateVersionID: preConfigVersion.TemplateVersionID,
		TemplateName:      preConfigVersion.TemplateName,
		TemplateNamespace: preConfigVersion.TemplateNamespace,
		ReplaceDigest:     preConfigVersion.ReplaceDigest,
	}
	if _, err := model.CreateConfigVersion(newConfigVersion, tx); err != nil {
		alog.Errorf("RollbackConfig: create config version err: %v", err)
		tx.Rollback()
		return err
	}
	if _, err := cm.storage.StoreConfig(newConfigVersion.SiteID, newConfigVersion.Namespace, newConfigVersion.Name, fmt.Sprintf("%v", newConfigVersion.Version), nil, data); err != nil {
		alog.Errorf("RollbackConfig: store config data err: %v", err)
		tx.Rollback()
		return err
	}
	if _, err := cm.storage.StoreReplacement(newConfigVersion.SiteID, newConfigVersion.Namespace, newConfigVersion.Name, fmt.Sprintf("%v", newConfigVersion.Version), nil, replaceData); err != nil {
		alog.Errorf("RollbackConfig: store config replace data err: %v", err)
		tx.Rollback()
		return err
	}
	// 4. update config current
	configCurrent.LatestConfigVersionID = newConfigVersion.ID
	if err := model.UpdateConfigCurrent(configCurrent, tx); err != nil {
		alog.Errorf("RollbackConfig: update config current err: %v", err)
		tx.Rollback()
		return err
	}
	// 5. update config info status
	configInfo.Status = model.ConfigStatusUpdated
	if err := model.UpdateConfigInfo(configInfo, tx); err != nil {
		alog.Errorf("RollbackConfig: update config info status err: %v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// DeleteConfig release a config
func (cm *configManager) DeleteConfig(id uint64) error {
	// TODO: unrelease config

	return nil
}

// ImportConfigs import configs from other config system
func (cm *configManager) ImportConfigs(siteID, desc, creator string, configType model.ConfigType, source model.ConfigSource, addr string, file multipart.File) error {
	if siteID == "" {
		return fmt.Errorf("site_id can not be null")
	}
	switch source {
	case model.ConfigSourceImage:
		return cm.importFromImage(siteID, desc, creator, configType, addr)
	case model.ConfigSourceTar:
		return cm.importFromTar(siteID, desc, creator, configType, file)
	default:
		return fmt.Errorf("invalid import source")
	}
}

// GetDefaultImportAddress return default import address
func (cm *configManager) GetDefaultImportAddress(siteID string, configType model.ConfigType, source model.ConfigSource) (string, error) {
	if siteID == "" {
		return "", fmt.Errorf("site_id can not be null")
	}
	switch source {
	case model.ConfigSourceImage:
		return fmt.Sprintf("harbor.xxxxx.cn/online-pipeline/camerainfos/%v:1.0.0", siteID), nil
	default:
		return "", nil
	}
}

// GetConfigDetail get config data
func (cm *configManager) GetConfigDetail(id uint64) (*model.ConfigInfo, error) {

	config, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("GetConfigDetail: get config info err: %v", err)
		return nil, err
	}

	file, templateFile, replaceFile, err := cm.visitConfigData(config)
	if err != nil {
		alog.Errorf("GetConfigDetail: get config data err: %v", err)
		return nil, err
	}
	if config.Format == model.ConfigFormatJPG {
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return nil, err
		}
		config.Data = base64.StdEncoding.EncodeToString(buf)
	} else {
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, file); err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return nil, err
		}
		config.Data = buf.String()
	}

	if templateFile != nil {
		templateBuf := new(strings.Builder)
		if _, err := io.Copy(templateBuf, templateFile); err != nil {
			alog.Errorf("visitConfigData: get template data error: %v", err)
			return nil, err
		}
		config.TemplateData = templateBuf.String()
	}

	if replaceFile != nil {
		replaceBuf := new(strings.Builder)
		if _, err := io.Copy(replaceBuf, replaceFile); err != nil {
			alog.Errorf("visitConfigData: get replace data error: %v", err)
			return nil, err
		}
		config.ReplaceData = replaceBuf.String()
	}

	return config, nil
}

// GetConfigInstances get config instances
func (cm *configManager) GetConfigInstances(id uint64) ([]*model.ConfigInstance, error) {

	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("GetConfigInstances: get config info err: %v", err)
		return nil, err
	}

	return cm.getConfigInstances(configInfo)
}

// CountChangeHistory count the change history of a config
func (cm *configManager) CountChangeHistory(query string) (int64, error) {

	return model.CountConfigVersions(query)
}

// ListChangeHistory get list of change history belongs to the config
func (cm *configManager) ListChangeHistory(query string, order []string, offset, limit int) ([]*model.ConfigVersion, error) {
	configVersions, err := model.ListConfigVersion(query, order, offset, limit)
	if err != nil {
		alog.Errorf("ListChangeHistory: list config version err: %v", err)
		return nil, err
	}
	return configVersions, nil
}

// GetChangeHistoryDetail get updated data and diff of one change history
func (cm *configManager) GetChangeHistoryDetail(id uint64, configVersionID uint64) (data string, diff string, meta string, err error) {

	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: get config info err: %v", err)
		return "", "", "", err
	}
	configVersion, err := model.GetConfigVersionByID(configVersionID)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: get config version err: %v", err)
		return "", "", "", err
	}

	file, _, _, err := cm.visitConfigVersionData(configVersion)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: get config data err: %v", err)
		return "", "", "", err
	}
	if file != nil {
		if configInfo.Format == model.ConfigFormatJPG {
			buf, err := ioutil.ReadAll(file)
			if err != nil {
				alog.Errorf("visitConfigData: get config data error: %v", err)
				return "", "", "", err
			}
			data = base64.StdEncoding.EncodeToString(buf)
		} else {
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, file); err != nil {
				alog.Errorf("visitConfigData: get config data error: %v", err)
				return "", "", "", err
			}
			data = buf.String()
		}
	}

	if configVersion.Version < 2 {
		return data, data, "", nil
	}
	preConfigVersion, err := model.GetConfigVersionByConfigInfoIDAndVersion(id, configVersion.Version-1)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: get pre config version err: %v", err)
		return "", "", "", err
	}

	// diff meta
	meta = cm.diffConfigMeta(configVersion, preConfigVersion)

	// diff data
	if configVersion.Digest == preConfigVersion.Digest {
		return data, "", meta, nil
	}
	preData := ""
	preFile, _, _, err := cm.visitConfigVersionData(preConfigVersion)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: get pre config version data err: %v", err)
		return "", "", "", err
	}
	if preFile != nil {
		if configInfo.Format == model.ConfigFormatJPG {
			buf, err := ioutil.ReadAll(preFile)
			if err != nil {
				alog.Errorf("visitConfigData: get config data error: %v", err)
				return "", "", "", err
			}
			preData = base64.StdEncoding.EncodeToString(buf)
		} else {
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, preFile); err != nil {
				alog.Errorf("visitConfigData: get config data error: %v", err)
				return "", "", "", err
			}
			preData = buf.String()
		}
	}

	diff, err = cm.diffConfigData(data, preData, configInfo.Format)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: diff config data err: %v", err)
		return "", "", "", err
	}

	return data, diff, meta, nil
}

// CountReleaseHistory count the release history of a config
func (cm *configManager) CountReleaseHistory(query string) (int64, error) {

	return model.CountReleases(query)
}

// ListReleaseHistory get list of release history belongs to the config
func (cm *configManager) ListReleaseHistory(query string, order []string, offset, limit int) ([]*model.Release, error) {
	releases, err := model.ListRelease(query, order, offset, limit)
	if err != nil {
		alog.Errorf("ListReleaseHistory: list release err: %v", err)
		return nil, err
	}

	return releases, nil
}

// GetReleaseHistoryDetail get updated data and diff of one release history
func (cm *configManager) GetReleaseHistoryDetail(id uint64, releaseID uint64) (data string, diff string, meta string, err error) {

	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get config info err: %v", err)
		return "", "", "", err
	}
	release, err := model.GetReleaseByID(releaseID)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get release err: %v", err)
		return "", "", "", err
	}
	configVersion, err := model.GetConfigVersionByID(release.ConfigVersionID)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get config version err: %v", err)
		return "", "", "", err
	}

	file, _, _, err := cm.visitConfigVersionData(configVersion)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get config data err: %v", err)
		return "", "", "", err
	}
	if configInfo.Format == model.ConfigFormatJPG {
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return "", "", "", err
		}
		data = base64.StdEncoding.EncodeToString(buf)
	} else {
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, file); err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return "", "", "", err
		}
		data = buf.String()
	}

	if release.Version < 2 {
		return data, data, "", nil
	}
	preRelease, err := model.GetReleaseByConfigInfoIDAndVersion(id, release.Version-1)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get pre release err: %v", err)
		return "", "", "", err
	}

	preConfigVersion, err := model.GetConfigVersionByID(preRelease.ConfigVersionID)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get pre config version err: %v", err)
		return "", "", "", err
	}

	// diff meta
	meta = cm.diffConfigMeta(configVersion, preConfigVersion)

	// diff data
	if configVersion.Digest == preConfigVersion.Digest {
		return data, "", meta, nil
	}
	preData := ""
	preFile, _, _, err := cm.visitConfigVersionData(preConfigVersion)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: get pre config version data err: %v", err)
		return "", "", "", err
	}
	if configInfo.Format == model.ConfigFormatJPG {
		buf, err := ioutil.ReadAll(preFile)
		if err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return "", "", "", err
		}
		preData = base64.StdEncoding.EncodeToString(buf)
	} else {
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, preFile); err != nil {
			alog.Errorf("visitConfigData: get config data error: %v", err)
			return "", "", "", err
		}
		preData = buf.String()
	}

	diff, err = cm.diffConfigData(data, preData, configInfo.Format)
	if err != nil {
		alog.Errorf("GetReleaseHistoryDetail: diff config data err: %v", err)
		return "", "", "", err
	}

	return data, diff, meta, nil
}

// ReloadConfig reload template to the config
func (cm *configManager) ReloadTemplate(id uint64) error {

	configInfo, err := model.GetConfigInfoByID(id)
	if err != nil {
		alog.Errorf("ReloadTemplate: get config info err: %v", err)
		return err
	}
	if configInfo.Format != model.ConfigFormatProperties {
		alog.Errorf("ReloadTemplate: unsupport config format")
		return err
	}

	renderConfig, err := cm.renderClusterEnv(*configInfo)
	if err != nil {
		alog.Errorf("ReloadTemplate: render cluster env err: %v", err)
		return err
	}
	_, _, replaceFile, err := cm.visitConfigData(renderConfig)
	if err != nil {
		alog.Errorf("ReloadTemplate: visit config data err: %v", err)
		return err
	}
	replaceData := ""
	if replaceFile != nil {
		replaceBuf := new(strings.Builder)
		if _, err := io.Copy(replaceBuf, replaceFile); err != nil {
			alog.Errorf("visitConfigData: get replace data error: %v", err)
			return err
		}
		replaceData = replaceBuf.String()
	}

	templateVersion, err := cm.getConfigTemplate(renderConfig)
	_, tmpFile, err := cm.storage.VisitTemplate(templateVersion.Namespace, templateVersion.Name, fmt.Sprintf("%v", templateVersion.Version))
	if err != nil {
		alog.Errorf("visitConfigData: get config data from file db error: %v", err)
		return err
	}
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, tmpFile); err != nil {
		alog.Errorf("visitConfigData: get config data error: %v", err)
		return err
	}
	templateData := buf.String()

	configInfo.TemplateVersionID = templateVersion.ID
	configInfo.TemplateData = templateData
	configInfo.ReplaceData = replaceData

	return cm.UpdateConfig(configInfo)
}

// AddTemplate add new template by uploading or writing on page
func (cm *configManager) AddTemplate(template *model.Template) error {

	if valid, err := cm.checkTemplateValidity(template); !valid {
		return err
	}

	tx := db.NewTransaction()
	// 1. create or find ns
	if _, err := model.FindOrCreateTemplateNamespace(template.Namespace, template.CreatedBy, tx); err != nil {
		alog.Errorf("AddTemplate find or create ns err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. create template
	template.Status = "created"
	if _, err := model.CreateTemplate(template, tx); err != nil {
		alog.Errorf("AddTemplate create template err: %v", err)
		tx.Rollback()
		return err
	}
	// 3. put data to file db
	digest, err := cm.storeTemplateData(template, 1)
	if err != nil {
		alog.Errorf("AddTemplate store data err: %v", err)
		tx.Rollback()
		return err
	}
	// 4. create new template version
	templateVersion := &model.TemplateVersion{
		Name:       template.Name,
		Namespace:  template.Namespace,
		Version:    1,
		Desc:       template.Desc,
		CreatedBy:  template.CreatedBy,
		Group:      template.Group,
		Digest:     digest,
		Status:     "created",
		TemplateID: template.ID,
	}
	if _, err := model.CreateTemplateVersion(templateVersion, tx); err != nil {
		alog.Errorf("AddTemplate create template version err: %v", err)
		tx.Rollback()
		return err
	}
	// 5. update template status
	template.LatestVersionID = templateVersion.ID
	if err := model.UpdateTemplate(template, tx); err != nil {
		alog.Errorf("AddTemplate update template status err: %v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// CountTemplateNamespaces return the number of template namespace
func (cm *configManager) CountTemplateNamespaces(query string) (int64, error) {

	return model.CountTemplateNamespace(query)
}

// ListTemplateNamespaces return the list of template namespace
func (cm *configManager) ListTemplateNamespaces(query string, orders []string, offset, limit int) ([]*model.TemplateNamespace, error) {

	templateNs, err := model.ListTemplateNamespace(query, orders, offset, limit)
	if err != nil {
		alog.Errorf("ListTemplateNamespaces: list err: %v", err)
		return nil, err
	}

	return templateNs, nil
}

// CountTemplates return the number of template
func (cm *configManager) CountTemplates(query string) (int64, error) {

	return model.CountTemplates(query)
}

// ListTemplates return the list of template
func (cm *configManager) ListTemplates(query string, orders []string, offset, limit int) ([]*model.Template, error) {

	templates, err := model.ListTemplate(query, orders, offset, limit)
	if err != nil {
		alog.Errorf("ListTemplates: list err: %v", err)
		return nil, err
	}

	return templates, nil
}

// UpdateTemplate update template
func (cm *configManager) UpdateTemplate(template *model.Template) error {

	if valid, err := cm.checkTemplateValidity(template); !valid {
		return err
	}

	newData := template.Data
	creator := template.CreatedBy
	template, err := model.GetTemplateByID(template.ID)
	if err != nil {
		alog.Errorf("UpdateTemplate: get template err: %v", err)
		return err
	}
	template.Data = newData
	template.CreatedBy = creator

	// begin a transaction
	tx := db.NewTransaction()
	// 1. get old template version
	oldTemplateVersion, err := model.GetTemplateVersionByID(template.LatestVersionID)
	if err != nil {
		alog.Errorf("UpdateTemplate: get old template version err: %v", err)
		tx.Rollback()
		return err
	}
	// 2. put data to file db
	digest, err := cm.storeTemplateData(template, oldTemplateVersion.Version+1)
	if err != nil {
		alog.Errorf("UpdateTemplate: store data err: %v", err)
		tx.Rollback()
		return err
	}
	// 3. create new version
	if cm.isTemplateNeedNewVersion(oldTemplateVersion, template, digest) {
		newTemplateVersion := &model.TemplateVersion{
			Name:       template.Name,
			Namespace:  template.Namespace,
			Version:    oldTemplateVersion.Version + 1,
			Desc:       template.Desc,
			CreatedBy:  template.CreatedBy,
			Group:      template.Group,
			Digest:     digest,
			Status:     "created",
			TemplateID: template.ID,
		}
		if _, err := model.CreateTemplateVersion(newTemplateVersion, tx); err != nil {
			alog.Errorf("UpdateTemplate: create new template version err: %v", err)
			tx.Rollback()
			return err
		}
		template.LatestVersionID = newTemplateVersion.ID
	}
	// 4. update template
	if err := model.UpdateTemplate(template, tx); err != nil {
		alog.Errorf("UpdateTemplate: update template err: %v", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// DeleteTemplate release a config
func (cm *configManager) DeleteTemplate(id uint64) error {
	// TODO: Consider cascading relationships
	return nil
}

// GetTemplateDetail get template data
func (cm *configManager) GetTemplateDetail(id uint64) (*model.Template, error) {

	template, err := model.GetTemplateByID(id)
	if err != nil {
		alog.Errorf("GetTemplateDetail: get template err: %v", err)
		return nil, err
	}

	data, err := cm.visitTemplateData(template)
	if err != nil {
		alog.Errorf("GetTemplateDetail: get template data err: %v", err)
		return nil, err
	}

	template.Data = data

	return template, nil
}

// GetTemplateDetailByNameAndNamespace get template data by name and namespace
func (cm *configManager) GetTemplateDetailByNameAndNamespace(name, namespace, siteID string) (*model.Template, error) {

	env, err := model2.GetClusterEnvBySiteID(siteID)
	if err != nil {
		return nil, err
	}
	if env != "" {
		envMap := make(map[string]string)
		if err := json.Unmarshal([]byte(env), &envMap); err != nil {
			return nil, err
		}
		if strings.HasPrefix(namespace, "${") && strings.HasSuffix(namespace, "}") {
			envKey := strings.TrimSuffix(strings.TrimPrefix(namespace, "${"), "}")
			if v, ok := envMap[envKey]; ok {
				namespace = v
			}
		}
	}

	template, err := model.GetTemplateByNameNamespace(name, namespace)
	if err != nil {
		alog.Errorf("GetTemplateDetail: get template err: %v", err)
		return nil, err
	}

	data, err := cm.visitTemplateData(template)
	if err != nil {
		alog.Errorf("GetTemplateDetail: get template data err: %v", err)
		return nil, err
	}

	template.Data = data

	return template, nil
}

// GetTemplateOption get the select option of template
func (cm *configManager) GetTemplateOption(siteID string) (map[string][]string, error) {
	templates, err := model.ListAllTemplate()
	if err != nil {
		alog.Errorf("GetTemplateOption: list template err: %v", err)
		return nil, err
	}

	options := make(map[string][]string)
	for _, template := range templates {
		if _, ok := options[template.Namespace]; !ok {
			options[template.Namespace] = make([]string, 0)
		}
		options[template.Namespace] = append(options[template.Namespace], template.Name)
	}

	env, err := model2.GetClusterEnvBySiteID(siteID)
	if err != nil {
		return nil, err
	}
	alog.Infof("env is: %+v", env)
	if env == "" {
		return options, nil
	}

	envMap := make(map[string]string)
	if err := json.Unmarshal([]byte(env), &envMap); err != nil {
		return nil, err
	}

	alog.Infof("env map is: %+v", envMap)
	for k, v := range envMap {
		if strings.HasPrefix(k, "CONFIG_") {
			if ops, ok := options[v]; ok {
				ns := "${" + k + "}"
				options[ns] = ops
			}
		}
	}

	return options, nil
}

// CountTemplateChangeHistory count the change history of a template
func (cm *configManager) CountTemplateChangeHistory(query string) (int64, error) {
	return model.CountTemplateVersions(query)
}

// ListTemplateChangeHistory get list of change history belongs to the template
func (cm *configManager) ListTemplateChangeHistory(query string, order []string, offset, limit int) ([]*model.TemplateVersion, error) {
	templateVersions, err := model.ListTemplateVersion(query, order, offset, limit)
	if err != nil {
		alog.Errorf("ListTemplateChangeHistory: list template version err: %v", err)
		return nil, err
	}
	return templateVersions, nil
}

// GetTemplateHistoryDetail get updated data and diff of one change template
func (cm *configManager) GetTemplateHistoryDetail(id uint64, versionID uint64) (diff string, err error) {

	templateVersion, err := model.GetTemplateVersionByID(versionID)
	if err != nil {
		alog.Errorf("GetTemplateHistoryDetail: get template version err: %v", err)
		return "", err
	}

	cur, err := cm.visitTemplateVersionData(templateVersion)
	if err != nil {
		alog.Errorf("GetTemplateHistoryDetail: get cur template diff err: %v", err)
		return "", err
	}

	if templateVersion.Version < 2 {
		return cur, nil
	}
	preVersion, err := model.GetTemplateVersionByTemplateIDAndVersion(id, templateVersion.Version-1)
	if err != nil {
		alog.Errorf("GetTemplateHistoryDetail: get pre template version err: %v", err)
		return "", err
	}

	// diff diff
	if templateVersion.Digest == preVersion.Digest {
		return "", nil
	}
	pre, err := cm.visitTemplateVersionData(preVersion)
	if err != nil {
		alog.Errorf("GetTemplateHistoryDetail: get pre template diff err: %v", err)
		return "", err
	}

	diff, err = cm.diffConfigData(cur, pre, model.ConfigFormatProperties)
	if err != nil {
		alog.Errorf("GetChangeHistoryDetail: diff config diff err: %v", err)
		return "", err
	}

	return diff, nil
}

func (cm *configManager) checkConfigValidity(config *model.ConfigInfo) (bool, error) {

	if config == nil {
		return false, fmt.Errorf("config entity can not be nil ")
	}
	if config.Name == "" {
		return false, fmt.Errorf("config name can not be nil ")
	}
	if config.Namespace == "" {
		return false, fmt.Errorf("config namespace can not be nil ")
	}
	if config.SiteID == "" {
		return false, fmt.Errorf("config site_id can not be nil ")
	}
	if config.CreatedBy == "" {
		return false, fmt.Errorf("config creator can not be nil ")
	}
	if config.Format == "" {
		return false, fmt.Errorf("config format can not be nil ")
	}

	return true, nil
}

func (cm *configManager) checkTemplateValidity(template *model.Template) (bool, error) {

	if template == nil {
		return false, fmt.Errorf("template entity can not be nil ")
	}
	if template.Name == "" {
		return false, fmt.Errorf("template name can not be nil ")
	}
	if template.Namespace == "" {
		return false, fmt.Errorf("template namespace can not be nil ")
	}
	if template.CreatedBy == "" {
		return false, fmt.Errorf("template creator can not be nil ")
	}

	return true, nil
}

func (cm *configManager) importFromImage(siteID, desc string, creator string, configType model.ConfigType, addr string) error {

	startAt := time.Now()
	alog.Infof("Start Import form image: %v at %v", addr, startAt.Format("2006-01-02 15:04:05"))

	dockerCli, err := docker.NewClient()
	if err != nil {
		alog.Errorf("importFromImage: create docker client error: %v", err)
		return err
	}

	distDir := filepath.Join(tmpDir, fmt.Sprintf("%v", time.Now().Unix()))
	if err := os.MkdirAll(distDir, 0777); err != nil {
		alog.Errorf("importFromImage: create dir error: %v", err)
		return err
	}
	defer os.RemoveAll(distDir)

	switch configType {
	case model.ConfigCameraInfo:
		if err := dockerCli.CopyFromImage(addr, cameraInfoDir, distDir); err != nil {
			alog.Errorf("importFromImage: copy from image error: %v", err)
			return err
		}
	case model.ConfigStoreInfo:
		if err := dockerCli.CopyFromImage(addr, storeInfoDir, distDir); err != nil {
			alog.Errorf("importFromImage: copy from image error: %v", err)
			return err
		}
	}
	alog.Infof("Start create config from dir: %v", distDir)
	if err := cm.createConfigFromDir(distDir, siteID, desc, creator); err != nil {
		alog.Errorf("importFromImage: create config error: %v", err)
		return err
	}

	finishAt := time.Now()
	alog.Infof("Finished create config from image: %v, take time: %v second", addr, finishAt.Sub(startAt).Seconds())

	return nil
}

func (cm *configManager) importFromTar(siteID, desc, creator string, configType model.ConfigType, file multipart.File) error {

	startAt := time.Now()

	tr := tar.NewReader(file)
	hdr, err := tr.Next()
	if err != nil {
		alog.Errorf("importFromTar: read tar err: %v", err)
		return err
	}
	if !hdr.FileInfo().IsDir() {
		err := fmt.Errorf("invalid format")
		alog.Errorf("importFromTar: check tar err: %v", err)
		return err
	}
	namespace := fmt.Sprintf(strings.TrimSuffix(hdr.Name, "/"))
	if _, err := model.FindOrCreateConfigNamespace(namespace, siteID, creator, nil); err != nil {
		alog.Errorf("importFromTar: find or create ns err: %v", err)
		return err
	}

	configCount := 0
	for hdr, err := tr.Next(); err != io.EOF; hdr, err = tr.Next() {
		if err != nil {
			alog.Errorf("importFromTar: read tar err: %v", err)
			return err
		}
		fi := hdr.FileInfo()
		if fi.IsDir() {
			continue
		}
		name := strings.TrimPrefix(hdr.Name, namespace+"/")

		if err := cm.createOrUpdateConfig(name, namespace, siteID, desc, creator, tr); err != nil {
			alog.Errorf("importFromTar: create config for %v err: %v", name, err)
			return err
		}

		configCount++
	}

	alog.Infof("importFromTar: create file num: %v", configCount)
	finishAt := time.Now()
	alog.Infof("Finished create config from tar, take time: %v second", finishAt.Sub(startAt).Seconds())

	return nil
}

func (cm *configManager) createConfigFromDir(dir string, siteID string, desc, creator string) error {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		alog.Errorf("createConfigFromDir: read dir err: %v", err)
		return err
	}
	if len(fs) != 1 {
		err := fmt.Errorf("invalid dir format")
		alog.Errorf("createConfigFromDir: read dir err: %v", err)
		return err
	}
	namespace := fs[0].Name()

	// 1. find or create namespace
	if _, err := model.FindOrCreateConfigNamespace(namespace, siteID, creator, nil); err != nil {
		alog.Errorf("createConfigFromDir: find or create ns err: %v", err)
		return err
	}
	// 2. walk dir/namespace
	src := filepath.Join(dir, namespace)
	configNum := 0
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, src+"/")
		if err := cm.createConfigFromFile(path, name, namespace, siteID, desc, creator); err != nil {
			return err
		}
		configNum++
		return nil

	}); err != nil {
		alog.Errorf("createConfigFromDir: create config err: %v", err)
		return err
	}

	alog.Infof("createConfigFromDir: create file num: %v", configNum)

	return nil
}

func (cm *configManager) createConfigFromFile(path, name, namespace, siteID, desc, creator string) error {

	content, err := os.Open(path)
	if err != nil {
		alog.Errorf("createConfigFromDir: open file %v err: %v", path, err)
		return err
	}
	defer content.Close()

	if err := cm.createOrUpdateConfig(name, namespace, siteID, desc, creator, content); err != nil {
		alog.Errorf("createConfigFromDir: create config %v err: %v", name, err)
		return err
	}

	return nil
}

func (cm *configManager) createOrUpdateConfig(name, namespace, siteID, desc, creator string, data io.Reader) error {

	tx := db.NewTransaction()
	// 1. get config info
	configInfo, err := model.GetConfigInfoByNamespaceNameSiteID(namespace, name, siteID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			configInfo = &model.ConfigInfo{
				Name:      name,
				Namespace: namespace,
				CreatedBy: creator,
				Desc:      desc,
				SiteID:    siteID,
				Format:    cm.getConfigFormat(name),
				Status:    model.ConfigStatusCreated,
			}
			if _, err := model.CreateConfigInfo(configInfo, tx); err != nil {
				alog.Errorf("createOrUpdateConfig: create config info for %v err: %v", configInfo.Name, err)
				tx.Rollback()
				return err
			}
		} else {
			alog.Errorf("createOrUpdateConfig: get config info for %v err: %v", configInfo.Name, err)
			tx.Rollback()
			return err
		}
	}
	// 2. get config current
	configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			configCurrent = &model.ConfigCurrent{
				ConfigInfoID:          configInfo.ID,
				LatestConfigVersionID: 0,
				LatestReleaseID:       0,
				PreReleaseID:          0,
				Status:                "created",
			}
			if _, err := model.CreateConfigCurrent(configCurrent, tx); err != nil {
				alog.Errorf("createOrUpdateConfig: create config current for %v err: %v", configInfo.Name, err)
				tx.Rollback()
				return err
			}
		} else {
			alog.Errorf("createOrUpdateConfig: get config current for %v err: %v", configInfo.Name, err)
			tx.Rollback()
			return err
		}
	}
	// 3. get latest config version digest
	oldDigest := ""
	oldVersion := uint64(0)
	if configCurrent.LatestConfigVersionID != 0 {
		configVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
		if err != nil {
			alog.Errorf("createOrUpdateConfig: get config version for %v err: %v", configInfo.Name, err)
			tx.Rollback()
			return err
		}
		oldDigest = configVersion.Digest
	}
	// 4. store data
	digest, err := cm.storage.StoreConfig(configInfo.SiteID, configInfo.Namespace, configInfo.Name, fmt.Sprintf("%v", oldVersion+1), nil, data)
	if err != nil {
		alog.Errorf("createOrUpdateConfig: store data for %v err: %v", configInfo.Name, err)
		tx.Rollback()
		return err
	}
	if oldDigest == digest {
		return nil
	}
	// 5. create new config version
	status := model.ConfigVersionUpdate
	if oldVersion == 0 {
		status = model.ConfigVersionCreate
	}
	newConfigVersion := &model.ConfigVersion{
		Name:         configInfo.Name,
		Namespace:    configInfo.Namespace,
		Version:      oldVersion + 1,
		Desc:         configInfo.Desc,
		CreatedBy:    configInfo.CreatedBy,
		SiteID:       configInfo.SiteID,
		Digest:       digest,
		Status:       status,
		ConfigInfoID: configInfo.ID,
	}
	if _, err := model.CreateConfigVersion(newConfigVersion, tx); err != nil {
		alog.Errorf("createOrUpdateConfig: create config version for %v err: %v", configInfo.Name, err)
		tx.Rollback()
		return err
	}
	// 6. update config status
	if configInfo.Status != model.ConfigStatusCreated {
		configInfo.Status = model.ConfigStatusUpdated
	}
	if err := model.UpdateConfigInfo(configInfo, tx); err != nil {
		alog.Errorf("createOrUpdateConfig: update config info for %v err: %v", configInfo.Name, err)
		tx.Rollback()
		return err
	}
	configCurrent.LatestConfigVersionID = newConfigVersion.ID
	if err := model.UpdateConfigCurrent(configCurrent, tx); err != nil {
		alog.Errorf("createOrUpdateConfig: update config current for %v err: %v", configInfo.Name, err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (cm *configManager) releaseConfig(configInfo *model.ConfigInfo, tx *gorm.DB) (*model.Release, error) {

	if tx == nil {
		tx = db.Get()
	}

	// 1. get current status
	configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
	if err != nil {
		alog.Errorf("ReleaseConfig: get config current err: %v", err)
		return nil, err
	}
	configVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
	if err != nil {
		alog.Errorf("ReleaseConfig: get config version err: %v", err)
		return nil, err
	}
	// 2. get old release
	oldVersion := uint64(0)
	oldReleaseID := uint64(0)
	if configCurrent.LatestReleaseID != 0 {
		oldRelease, err := model.GetReleaseByID(configCurrent.LatestReleaseID)
		if err != nil {
			alog.Errorf("ReleaseConfig: get old release err: %v", err)
			return nil, err
		}
		if oldRelease.ConfigVersionID == configCurrent.LatestConfigVersionID {
			return nil, fmt.Errorf("this config has not new updates")
		}
		oldVersion = oldRelease.Version
		oldReleaseID = oldRelease.ID
	}
	// 3. create new release
	newRelease := &model.Release{
		Version:         oldVersion + 1,
		Desc:            configInfo.Desc,
		CreatedBy:       configInfo.CreatedBy,
		Status:          "created",
		ConfigInfoID:    configInfo.ID,
		ConfigVersionID: configCurrent.LatestConfigVersionID,
		Digest:          configVersion.Digest,
	}
	if _, err := model.CreateRelease(newRelease, tx); err != nil {
		alog.Errorf("ReleaseConfig: create release err: %v", err)
		return nil, err
	}
	// 4. update config current table
	configCurrent.LatestReleaseID = newRelease.ID
	configCurrent.PreReleaseID = oldReleaseID
	if err := model.UpdateConfigCurrent(configCurrent, tx); err != nil {
		alog.Errorf("ReleaseConfig: update config current err: %v", err)
		return nil, err
	}

	configInfo.Status = model.ConfigStatusReleased
	if err := model.UpdateConfigInfo(configInfo, tx); err != nil {
		alog.Errorf("ReleaseConfig: update config info status err: %v", err)
		return nil, err
	}

	return newRelease, nil
}

func (cm *configManager) getReleaseConfigDetail(configInfo *model.ConfigInfo) (data string, instances []*model.ConfigInstance, err error) {

	// 1. get current status
	configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
	if err != nil {
		alog.Errorf("getReleaseConfigDetail: get config current err: %v", err)
		return "", nil, err
	}
	// 2. get release data
	configVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
	if err != nil {
		alog.Errorf("getReleaseConfigDetail: get config version err: %v", err)
		return "", nil, err
	}
	cur, _, _, err := cm.visitConfigVersionData(configVersion)
	if err != nil {
		alog.Errorf("getReleaseConfigDetail: get current data err: %v", err)
		return "", nil, err
	}
	// 3. get old data
	var pre io.Reader
	if configCurrent.LatestReleaseID != 0 {
		oldRelease, err := model.GetReleaseByID(configCurrent.LatestReleaseID)
		if err != nil {
			alog.Errorf("getReleaseConfigDetail: get old release err: %v", err)
			return "", nil, err
		}
		oldConfigVersion, err := model.GetConfigVersionByID(oldRelease.ConfigVersionID)
		if err != nil {
			alog.Errorf("getReleaseConfigDetail: get old release version err: %v", err)
			return "", nil, err
		}
		pre, _, _, err = cm.visitConfigVersionData(oldConfigVersion)
		if err != nil {
			alog.Errorf("getReleaseConfigDetail: get pre data err: %v", err)
			return "", nil, err
		}
	}
	// 4. diff data
	if configInfo.Format == model.ConfigFormatJPG {
		buf, err := ioutil.ReadAll(cur)
		if err != nil {
			alog.Errorf("getReleaseConfigDetail: read config data error: %v", err)
			return "", nil, err
		}
		data = base64.StdEncoding.EncodeToString(buf)
	} else {
		curData := ""
		if cur != nil {
			curBuf := new(strings.Builder)
			if _, err := io.Copy(curBuf, cur); err != nil {
				alog.Errorf("getReleaseConfigDetail: read config data error: %v", err)
				return "", nil, err
			}
			curData = curBuf.String()
		}

		preData := ""
		if pre != nil {
			preBuf := new(strings.Builder)
			if _, err := io.Copy(preBuf, pre); err != nil {
				alog.Errorf("getReleaseConfigDetail: read config data error: %v", err)
				return "", nil, err
			}
			preData = preBuf.String()
		}

		data, err = cm.diffConfigData(curData, preData, configInfo.Format)
		if err != nil {
			alog.Errorf("getReleaseConfigDetail: diff config data error: %v", err)
			return "", nil, err
		}
	}
	// 5. get instances
	instances, err = cm.getConfigInstances(configInfo)
	if err != nil {
		alog.Errorf("getReleaseConfigDetail: get config instance error: %v", err)
		return "", nil, err
	}

	return data, instances, nil
}

func (cm *configManager) getConfigInstances(configInfo *model.ConfigInfo) ([]*model.ConfigInstance, error) {
	cluster, err := model2.GetClusterBySiteID(configInfo.SiteID)
	if err != nil {
		alog.Errorf("getConfigInstances: get cluster err: %v", err)
		return nil, err
	}
	instances, err := model.GetConfigInstanceByConfigID(configInfo.ID, cluster.ConnKey)
	if err != nil {
		alog.Errorf("getConfigInstances: get config instances err: %v", err)
		return nil, err
	}

	// check is sync
	for _, instance := range instances {
		configCurrent, err := model.GetConfigCurrentByConfigID(configInfo.ID)
		if err != nil {
			alog.Errorf("getConfigInstances: get config current %v err: %v", configInfo.Name, err)
			continue
		}
		release, err := model.GetReleaseByID(configCurrent.LatestReleaseID)
		if err != nil {
			alog.Errorf("getConfigInstances: get config release %v err: %v", configInfo.Name, err)
			continue
		}
		if release.Digest != instance.Digest {
			instance.IsSync = false
		} else {
			instance.IsSync = true
		}
	}

	if instances == nil {
		instances = []*model.ConfigInstance{}
	}
	return instances, nil
}

func (cm *configManager) getConfigFormat(name string) model.ConfigFormat {

	return formatMap[path.Ext(name)]
}

func (cm *configManager) getConfigTemplate(config *model.ConfigInfo) (*model.TemplateVersion, error) {
	if config.TemplateName == "" {
		return nil, fmt.Errorf("config template name can't be null")
	}
	if config.TemplateNamespace == "" {
		return nil, fmt.Errorf("config template namespace can't be null")
	}

	template, err := model.GetTemplateByNameNamespace(config.TemplateName, config.TemplateNamespace)
	if err != nil {
		return nil, fmt.Errorf("get template error: %v", err)
	}
	templateVersion, err := model.GetTemplateVersionByID(template.LatestVersionID)
	if err != nil {
		return nil, fmt.Errorf("get template version error: %v", err)
	}

	return templateVersion, nil
}

func (cm *configManager) storeConfigData(config *model.ConfigInfo, version uint64) (digest string, replaceDigest string, err error) {

	if config.Format == model.ConfigFormatProperties && config.TemplateData != "" {
		data, err := cm.mergeTemplateAndReplace(config.TemplateData, config.ReplaceData)
		if err != nil {
			return "", "", err
		}

		buf := strings.NewReader(config.ReplaceData)
		replaceDigest, err = cm.storage.StoreReplacement(config.SiteID, config.Namespace, config.Name, fmt.Sprintf("%v", version), nil, buf)
		if err != nil {
			return "", "", err
		}

		config.Data = data
	}
	buf := strings.NewReader(config.Data)
	digest, err = cm.storage.StoreConfig(config.SiteID, config.Namespace, config.Name, fmt.Sprintf("%v", version), nil, buf)
	if err != nil {
		return "", "", err
	}

	return digest, replaceDigest, nil
}

func (cm *configManager) visitConfigData(config *model.ConfigInfo) (data io.Reader, templateData io.Reader, replaceData io.Reader, err error) {

	configCurrent, err := model.GetConfigCurrentByConfigID(config.ID)
	if err != nil {
		alog.Errorf("visitConfigData: get config current error: %v", err)
		return nil, nil, nil, err
	}

	configVersion, err := model.GetConfigVersionByID(configCurrent.LatestConfigVersionID)
	if err != nil {
		alog.Errorf("visitConfigData: get config version error: %v", err)
		return nil, nil, nil, err
	}

	return cm.visitConfigVersionData(configVersion)
}

func (cm *configManager) visitConfigVersionData(configVersion *model.ConfigVersion) (data io.Reader, templateData io.Reader, replaceData io.Reader, err error) {
	_, file, err := cm.storage.VisitConfig(configVersion.SiteID, configVersion.Namespace, configVersion.Name, fmt.Sprintf("%v", configVersion.Version))
	if err != nil {
		alog.Errorf("visitConfigData: get config data from file db error: %v", err)
		return nil, nil, nil, err
	}
	data = file

	if configVersion.TemplateVersionID != 0 {

		templateVersion, err := model.GetTemplateVersionByID(configVersion.TemplateVersionID)
		if err != nil {
			alog.Errorf("visitConfigData: get template version error: %v", err)
			return nil, nil, nil, err
		}
		_, tempFile, err := cm.storage.VisitTemplate(templateVersion.Namespace, templateVersion.Name, fmt.Sprintf("%v", templateVersion.Version))
		if err != nil {
			alog.Errorf("visitConfigData: get template data error: %v", err)
			return nil, nil, nil, err
		}
		templateData = tempFile

		_, replaceFile, err := cm.storage.VisitReplacement(configVersion.SiteID, configVersion.Namespace, configVersion.Name, fmt.Sprintf("%v", configVersion.Version))
		if err != nil {
			alog.Errorf("visitConfigData: get replace data from file db error: %v", err)
			return nil, nil, nil, err
		}
		replaceData = replaceFile
	}

	return data, templateData, replaceData, nil
}

func (cm *configManager) storeTemplateData(template *model.Template, version uint64) (digest string, err error) {

	buf := strings.NewReader(strings.TrimSuffix(template.Data, "\n"))
	digest, err = cm.storage.StoreTemplate(template.Namespace, template.Name, fmt.Sprintf("%v", version), nil, buf)
	if err != nil {
		alog.Errorf("storeTemplateData: store template data error: %v", err)
		return "", err
	}
	return digest, nil
}

func (cm *configManager) visitTemplateData(template *model.Template) (string, error) {

	templateVersion, err := model.GetTemplateVersionByID(template.LatestVersionID)
	if err != nil {
		alog.Errorf("visitTemplateData: get template version error: %v", err)
		return "", err
	}

	return cm.visitTemplateVersionData(templateVersion)
}

func (cm *configManager) visitTemplateVersionData(templateVersion *model.TemplateVersion) (data string, err error) {
	_, file, err := cm.storage.VisitTemplate(templateVersion.Namespace, templateVersion.Name, fmt.Sprintf("%v", templateVersion.Version))
	if err != nil {
		alog.Errorf("visitTemplateVersionData: get template data from file db error: %v", err)
		return "", err
	}
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, file); err != nil {
		alog.Errorf("visitTemplateVersionData: get template data error: %v", err)
		return "", err
	}
	data = buf.String()

	return data, nil
}

func (cm *configManager) mergeTemplateAndReplace(templateData, replaceData string) (string, error) {

	if templateData == "" && replaceData == "" {
		return "", nil
	}

	if templateData == "" {
		return replaceData, nil
	}

	if replaceData == "" {
		return templateData, nil
	}
	templateProperties, err := properties.LoadString(strings.TrimSuffix(templateData, "\n"))
	if err != nil {
		return "", err
	}
	replaceProperties, err := properties.LoadString(strings.TrimSuffix(replaceData, "\n"))
	if err != nil {
		return "", err
	}

	replaceKeys := replaceProperties.Keys()

	for _, k := range replaceKeys {

		v, _ := replaceProperties.Get(k)

		_, _, err := templateProperties.Set(k, v)
		if err != nil {
			return "", err
		}
	}
	return templateProperties.String(), nil
}

func (cm *configManager) renderClusterEnv(config model.ConfigInfo) (*model.ConfigInfo, error) {

	env, err := model2.GetClusterEnvBySiteID(config.SiteID)
	if err != nil {
		return nil, err
	}
	if env == "" {
		return &config, nil
	}
	envMap := make(map[string]string)
	//FIXME: format env
	if err := json.Unmarshal([]byte(env), &envMap); err != nil {
		return nil, err
	}

	if strings.HasPrefix(config.TemplateNamespace, "${") && strings.HasSuffix(config.TemplateNamespace, "}") {
		envKey := strings.TrimSuffix(strings.TrimPrefix(config.TemplateNamespace, "${"), "}")
		if v, ok := envMap[envKey]; ok {
			config.TemplateNamespace = v
		}
	}

	// TODO: maybe more parameter need dynamic render

	return &config, nil
}

func (cm *configManager) checkIsTemplateChange(config *model.ConfigInfo) error {

	if config.TemplateVersionID == 0 || config.TemplateName == "" || config.TemplateNamespace == "" {
		config.IsTemplateChange = false
		return nil
	}

	renderConfig, err := cm.renderClusterEnv(*config)
	if err != nil {
		return err
	}

	templateVersion, err := cm.getConfigTemplate(renderConfig)
	if err != nil {
		return err
	}

	if config.TemplateVersionID != templateVersion.ID {
		config.IsTemplateChange = true
	} else {
		config.IsTemplateChange = false
	}

	return nil
}

func (cm *configManager) isConfigNeedNewVersion(old *model.ConfigVersion, new *model.ConfigInfo, digest, replaceDigest string) bool {
	if old.Namespace == new.Namespace &&
		old.Name == new.Name &&
		old.Desc == new.Desc &&
		old.Digest == digest &&
		old.ReplaceDigest == replaceDigest &&
		old.TemplateVersionID == new.TemplateVersionID {
		return false
	}

	return true
}

func (cm *configManager) isTemplateNeedNewVersion(old *model.TemplateVersion, new *model.Template, digest string) bool {
	if old.Namespace == new.Namespace &&
		old.Name == new.Name &&
		old.Desc == new.Desc &&
		old.Digest == digest {
		return false
	}

	return true
}

func (cm *configManager) diffConfigMeta(cur, pre *model.ConfigVersion) string {
	result := ""
	if cur.Namespace != pre.Namespace {
		result = result + fmt.Sprintf("命名空间: %v -> %v", pre.Namespace, cur.Namespace) + "\n"
	}
	if cur.Name != pre.Name {
		result = result + fmt.Sprintf("名称: %v -> %v", pre.Name, cur.Name) + "\n"
	}
	if cur.Desc != pre.Desc {
		result = result + fmt.Sprintf("描述: %v -> %v", pre.Desc, cur.Desc) + "\n"
	}
	if cur.TemplateNamespace != pre.TemplateNamespace {
		result = result + fmt.Sprintf("模板命名空间: %v -> %v", pre.TemplateNamespace, cur.TemplateNamespace) + "\n"
	}
	if cur.TemplateName != pre.TemplateName {
		result = result + fmt.Sprintf("模板名称: %v -> %v", pre.TemplateName, cur.TemplateName) + "\n"
	}

	return result
}

func (cm *configManager) diffConfigData(cur, pre string, format model.ConfigFormat) (string, error) {

	// diff image config return pre data
	if format == model.ConfigFormatJPG {
		return pre, nil
	}

	return diff.Diff(pre, cur), nil

}

func (cm *configManager) packConfigs(configInfos []*model.ConfigInfo, file string, tx *gorm.DB) error {

	if tx == nil {
		tx = db.Get()
	}

	outFile, err := os.Create(file)
	if err != nil {
		alog.Errorf("packConfigs: create temp tar err: %v", err)
		return err
	}
	defer outFile.Close()

	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	filePaths := map[string]string{}

	for _, configInfo := range configInfos {

		release, err := cm.releaseConfig(configInfo, tx)
		if err != nil {
			alog.Errorf("packConfigs: release config for %v err: %v", configInfo.Name, err)
			return err
		}
		configVersion, err := model.GetConfigVersionByID(release.ConfigVersionID)
		if err != nil {
			alog.Errorf("packConfigs: get config version for %v err: %v", configInfo.Name, err)
			return err
		}
		fullpath := cm.storage.GetFilePath(configInfo.SiteID, configVersion.Namespace, configVersion.Name, fmt.Sprintf("%v", configVersion.Version))
		filePaths[fullpath] = configVersion.Name
	}

	if err := cm.storage.PackRawFiles(tarWriter, filePaths); err != nil {
		alog.Errorf("packConfigs: get config package err: %v", err)
		return err
	}

	return nil
}

func (cm *configManager) getSiteIDFromConnKey(connKey string) (string, error) {
	connInfos := strings.Split(connKey, apis.ConnectionSplit)
	if len(connInfos) != 2 {
		err := fmt.Errorf("invalid conn key: %v", connKey)
		alog.Errorf("AppCancelHandler: %v ", err)
		return "", err
	}

	return connInfos[0], nil
}

func (cm *configManager) getConnKeyFromSiteID(siteID string) (string, error) {

	cluster, err := model2.GetClusterBySiteID(siteID)
	if err != nil {
		return "", err
	}
	if cluster.ConnKey == "" {
		return "", fmt.Errorf("cluster miss")
	}

	return cluster.ConnKey, nil
}
