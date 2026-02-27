package bot

import (
	"fmt"
	"sync"

	"qq-farm-bot/internal/config"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

// Manager manages multiple bot instances.
type Manager struct {
	mu        sync.RWMutex
	instances map[int64]*Instance // accountID -> instance
	store     *store.Store
	cfg       *config.Config
}

func NewManager(s *store.Store, cfg *config.Config) *Manager {
	return &Manager{
		instances: make(map[int64]*Instance),
		store:     s,
		cfg:       cfg,
	}
}

// AutoStart starts all accounts with auto_start=true.
func (m *Manager) AutoStart() {
	accounts, err := m.store.ListAccounts()
	if err != nil {
		fmt.Printf("[Manager] 加载账号失败: %v\n", err)
		return
	}
	for _, a := range accounts {
		if a.AutoStart && a.Code != "" {
			acct := a
			if err := m.StartBot(&acct); err != nil {
				fmt.Printf("[Manager] 自动启动账号 #%d (%s) 失败: %v\n", a.ID, a.Name, err)
			}
		}
	}
}

func (m *Manager) StartBot(account *model.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst, ok := m.instances[account.ID]; ok && inst.IsRunning() {
		return fmt.Errorf("bot #%d already running", account.ID)
	}

	inst := NewInstance(account, m.cfg.GameServerURL, m.cfg.ClientVersion, m.store)
	if err := inst.Start(); err != nil {
		return err
	}
	m.instances[account.ID] = inst
	return nil
}

func (m *Manager) StopBot(accountID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	inst, ok := m.instances[accountID]
	if !ok {
		return fmt.Errorf("bot #%d not found", accountID)
	}
	inst.Stop()
	return nil
}

func (m *Manager) GetStatus(accountID int64) *model.BotStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	inst, ok := m.instances[accountID]
	if !ok {
		return &model.BotStatus{AccountID: accountID, Running: false}
	}
	return inst.Status()
}

func (m *Manager) GetAllStatus() []*model.BotStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statuses []*model.BotStatus
	for _, inst := range m.instances {
		statuses = append(statuses, inst.Status())
	}
	return statuses
}

func (m *Manager) GetInstance(accountID int64) *Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.instances[accountID]
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		inst.Stop()
	}
}
