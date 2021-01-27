package configserver

import (
	"mime/multipart"

	model "code.xxxxx.cn/platform/galaxy/pkg/manager/model/configserver"
)

// Manager define the manager of config server
type Manager interface {
	// AddConfig create a new config
	AddConfig(info *model.ConfigInfo) error
	// CountConfigNamespaces return the number of config namespace
	CountConfigNamespaces(query string) (int64, error)
	// ListConfigNamespaces return the list of config namespace
	ListConfigNamespaces(query string, orders []string, offset, limit int) ([]*model.ConfigNamespace, error)
	// CountConfigs return the number of config
	CountConfigs(query string) (int64, error)
	// ListConfigs return the list of config
	ListConfigs(query string, orders []string, offset, limit int) ([]*model.ConfigInfo, error)
	// UpdateConfig update a config
	UpdateConfig(info *model.ConfigInfo) error
	// ReleaseConfig release a config
	ReleaseConfig(id uint64, desc, creator string) error
	// ReleaseConfigDetail get release config detail
	ReleaseConfigDetail(id uint64) (string, []*model.ConfigInstance, error)
	// ReleaseNamespace
	ReleaseNamespace(siteID string, ns string, desc, creator string) error
	// ReleaseNamespaceDetail get release ns detail
	ReleaseNamespaceDetail(siteID string, ns string) ([]*model.ConfigAndInstance, error)
	// RollbackConfig rollback a config
	RollbackConfig(id uint64, versionID uint64, creator string) error
	// DeleteConfig delete a config
	DeleteConfig(id uint64) error
	// ImportConfigs import config from image or tar
	ImportConfigs(siteID, desc, creator string, configType model.ConfigType, source model.ConfigSource, addr string, file multipart.File) error
	// GetDefaultImportAddress get default import address
	GetDefaultImportAddress(siteID string, configType model.ConfigType, source model.ConfigSource) (string, error)
	// GetConfigDetail get config data and template data if it has
	GetConfigDetail(id uint64) (*model.ConfigInfo, error)
	// GetConfigInstances get config instances
	GetConfigInstances(id uint64) ([]*model.ConfigInstance, error)
	// CountChangeHistory count the change history of a config
	CountChangeHistory(query string) (int64, error)
	// ListChangeHistory get list of change history belongs to the config
	ListChangeHistory(query string, order []string, offset, limit int) ([]*model.ConfigVersion, error)
	// GetChangeHistoryDetail get updated data and diff of one change history
	GetChangeHistoryDetail(id uint64, versionID uint64) (string, string, string, error)
	// CountReleaseHistory count the release history of a config
	CountReleaseHistory(query string) (int64, error)
	// ListReleaseHistory get list of release history belongs to the config
	ListReleaseHistory(query string, order []string, offset, limit int) ([]*model.Release, error)
	// GetChangeHistoryDetail get updated data and diff of one release history
	GetReleaseHistoryDetail(id uint64, versionID uint64) (string, string, string, error)
	// ReloadTemplate reload the template of config to add a new config version
	ReloadTemplate(id uint64) error

	// AddTemplate create a new config template
	AddTemplate(info *model.Template) error
	// CountTemplateNamespaces return the number of template namespace
	CountTemplateNamespaces(query string) (int64, error)
	// ListTemplateNamespaces return the list of template namespace
	ListTemplateNamespaces(query string, orders []string, offset, limit int) ([]*model.TemplateNamespace, error)
	// CountTemplates return the number of tempate
	CountTemplates(query string) (int64, error)
	// ListTemplates return the list of tempate
	ListTemplates(query string, orders []string, offset, limit int) ([]*model.Template, error)
	// UpdateTemplate update a template
	UpdateTemplate(template *model.Template) error
	// DeleteTemplate delete a template
	DeleteTemplate(id uint64) error
	// GetTemplateDetail get template data
	GetTemplateDetail(id uint64) (*model.Template, error)
	// GetTemplateDetailByNameAndNamespace get template data by name and namespace
	GetTemplateDetailByNameAndNamespace(name, namespace, siteID string) (*model.Template, error)
	// GetTemplateOption get the select option of template
	GetTemplateOption(siteID string) (map[string][]string, error)
	// CountTemplateChangeHistory count the change history of a template
	CountTemplateChangeHistory(query string) (int64, error)
	// ListTemplateChangeHistory get list of change history belongs to the template
	ListTemplateChangeHistory(query string, order []string, offset, limit int) ([]*model.TemplateVersion, error)
	// GetTemplateHistoryDetail get updated data and diff of one change template
	GetTemplateHistoryDetail(id uint64, versionID uint64) (string, error)
}
