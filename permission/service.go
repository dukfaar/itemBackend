package permission

import (
	"sync"
	"sync/atomic"
	"time"
)

//TokenData represents an AccessToken
type TokenData struct {
	userID               string
	accessTokenExpiresAt time.Time
}

//Service handles all data needed to determine user permissions
type Service struct {
	tokenData          map[string]*TokenData
	userRoleData       map[string][]string
	rolePermissionData map[string][]string
	permissionData     map[string]string // id -> string

	userPermissionData             atomic.Value //map[string]map[string]bool
	userPermissionDataIsBuilt      bool
	userPermissionDataIsBuiltMutex sync.Mutex
}

//NewService creates a new Service
func NewService() *Service {
	return &Service{
		tokenData:                 make(map[string]*TokenData),
		userRoleData:              make(map[string][]string),
		rolePermissionData:        make(map[string][]string),
		permissionData:            make(map[string]string),
		userPermissionDataIsBuilt: false,
	}
}

//BuildUserPermissionData builds a map to determine which permissions a user has
//call this after changing data
func (m *Service) BuildUserPermissionData() {
	m.userPermissionDataIsBuiltMutex.Lock()
	defer m.userPermissionDataIsBuiltMutex.Unlock()
	if m.userPermissionDataIsBuilt == false {
		newUserPermissionData := make(map[string]map[string]bool)

		for userID := range m.userRoleData {
			userRoles := m.userRoleData[userID]
			newUserPermissionData[userID] = make(map[string]bool)

			for i := range userRoles {
				rolePermissions := m.rolePermissionData[userRoles[i]]
				for j := range rolePermissions {
					newUserPermissionData[userID][m.permissionData[rolePermissions[j]]] = true
				}
			}
		}
		m.userPermissionData.Store(newUserPermissionData)
		m.userPermissionDataIsBuilt = true
	}

}

//SetToken sets the value of a Token in the Service
func (m *Service) SetToken(accessToken string, userID string, expiresAt time.Time) {
	m.userPermissionDataIsBuiltMutex.Lock()
	defer m.userPermissionDataIsBuiltMutex.Unlock()
	m.tokenData[accessToken] = &TokenData{userID, expiresAt}
	m.userPermissionDataIsBuilt = false
}

//SetUser sets the value of a User in the Service
func (m *Service) SetUser(userID string, roles []string) {
	m.userPermissionDataIsBuiltMutex.Lock()
	defer m.userPermissionDataIsBuiltMutex.Unlock()
	m.userRoleData[userID] = roles
	m.userPermissionDataIsBuilt = false
}

//SetRole sets the value of a Role in the Service
func (m *Service) SetRole(roleID string, permissions []string) {
	m.userPermissionDataIsBuiltMutex.Lock()
	defer m.userPermissionDataIsBuiltMutex.Unlock()
	m.rolePermissionData[roleID] = permissions
	m.userPermissionDataIsBuilt = false
}

//SetPermission sets the value of a Permission in the Service
func (m *Service) SetPermission(permissionID string, name string) {
	m.userPermissionDataIsBuiltMutex.Lock()
	defer m.userPermissionDataIsBuiltMutex.Unlock()
	m.permissionData[permissionID] = name
	m.userPermissionDataIsBuilt = false
}

//TokenHasPermission checks if the user with the given accessToken has a certain permission
func (m *Service) TokenHasPermission(accessToken string, permissionName string) bool {
	token := m.tokenData[accessToken]
	if token == nil {
		return false
	}

	if token.accessTokenExpiresAt.Before(time.Now()) {
		return false
	}

	userPermissionData := m.userPermissionData.Load().(map[string]map[string]bool)
	return userPermissionData[token.userID][permissionName]
}
