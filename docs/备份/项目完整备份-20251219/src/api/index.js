import { createAlova } from 'alova'
import adapterFetch from 'alova/fetch'
import { ElMessage } from 'element-plus'
import router from '@/router'

// 创建 alova 实例
const alovaInstance = createAlova({
  baseURL: '/api',
  timeout: 10000,
  requestAdapter: adapterFetch(),
  cacheFor: null, // 禁用缓存，确保每次请求都获取最新数据

  // 请求前置钩子
  beforeRequest(method) {
    const token = localStorage.getItem('token')
    if (token) {
      method.config.headers.Authorization = `Bearer ${token}`
    }
  },

  // 响应处理
  responded: {
    onSuccess: async (response, method) => {
      // 先检查 401 错误，自动跳转登录页
      if (response.status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        router.push('/login')
        throw new Error('登录已过期，请重新登录')
      }

      const data = await response.json()
      if (response.ok) {
        return data
      }
      // 业务错误
      const msg = data?.message || data?.error || '请求失败'
      ElMessage.error(msg)
      throw new Error(msg)
    },
    onError: (error, method) => {
      ElMessage.error(error.message || '网络错误')
      throw error
    }
  }
})

// 封装请求方法，自动调用 send()
const Get = (url, config) => alovaInstance.Get(url, config).send()
const Post = (url, data, config) => alovaInstance.Post(url, data, config).send()
const Put = (url, data, config) => alovaInstance.Put(url, data, config).send()
const Delete = (url, config) => alovaInstance.Delete(url, config).send()

// API 方法封装
export default {
  // Auth
  getCaptcha: () => Get('/auth/captcha'),
  login: (data) => Post('/auth/login', data),
  register: (data) => Post('/auth/register', data),

  // Profile
  getProfile: () => Get('/profile'),
  updateProfile: (data) => Put('/profile', data),
  changePassword: (data) => Put('/profile/password', data),

  // Admin - Users
  getUsers: (params) => Get('/admin/users', { params }),
  getUser: (id) => Get(`/admin/users/${id}`),
  createUser: (data) => Post('/admin/users', data),
  updateUser: (id, data) => Put(`/admin/users/${id}`, data),
  deleteUser: (id) => Delete(`/admin/users/${id}`),

  // Admin - Accounts
  getAccountTypes: () => Get('/admin/accounts/types'),
  getAccounts: (params) => Get('/admin/accounts', { params }),
  getAccount: (id) => Get(`/admin/accounts/${id}`),
  createAccount: (data) => Post('/admin/accounts', data),
  updateAccount: (id, data) => Put(`/admin/accounts/${id}`, data),
  deleteAccount: (id) => Delete(`/admin/accounts/${id}`),
  updateAccountStatus: (id, data) => Put(`/admin/accounts/${id}/status`, data),

  // Admin - Account Groups
  getAccountGroups: (params) => Get('/admin/account-groups', { params }),
  getAllAccountGroups: () => Get('/admin/account-groups/all'),
  getAccountGroup: (id) => Get(`/admin/account-groups/${id}`),
  createAccountGroup: (data) => Post('/admin/account-groups', data),
  updateAccountGroup: (id, data) => Put(`/admin/account-groups/${id}`, data),
  deleteAccountGroup: (id) => Delete(`/admin/account-groups/${id}`),
  addAccountToGroup: (groupId, accountId) => Post(`/admin/account-groups/${groupId}/accounts`, { account_id: accountId }),
  removeAccountFromGroup: (groupId, accountId) => Delete(`/admin/account-groups/${groupId}/accounts/${accountId}`),

  // Admin - Request Logs (从 Redis 获取)
  getRequestLogs: (params) => Get('/admin/logs', { params }),
  getRequestLogSummary: (params) => Get('/admin/logs/summary', { params }),
  getAccountLoadStats: (params) => Get('/admin/logs/account-load', { params }),
  getAllUsageSummary: (params) => Get('/admin/logs/usage-summary', { params }),

  // Admin - User Usage Records (从 Redis 获取详细记录)
  getUserUsageRecords: (userId, params) => Get(`/admin/users/${userId}/usage/records`, { params }),

  // Admin - OAuth
  generateOAuthUrl: (platform, proxy) => Post('/admin/oauth/generate-url', { platform, proxy }),
  exchangeOAuthCode: (platform, code, session_id, proxy) => Post('/admin/oauth/exchange', { platform, code, session_id, proxy }),
  oauthByCookie: (platform, session_key, proxy) => Post('/admin/oauth/cookie-auth', { platform, session_key, proxy }),

  // Admin - Dashboard
  getDashboardStats: () => Get('/admin/dashboard/stats'),

  // Admin - System
  getSystemInfo: () => Get('/admin/system/info'),
  testProxy: (proxy) => Post('/admin/system/test-proxy', proxy),

  // Admin - Models
  getModels: (params) => Get('/admin/models', { params }),
  getModel: (id) => Get(`/admin/models/${id}`),
  createModel: (data) => Post('/admin/models', data),
  updateModel: (id, data) => Put(`/admin/models/${id}`, data),
  deleteModel: (id) => Delete(`/admin/models/${id}`),
  toggleModel: (id) => Put(`/admin/models/${id}/toggle`),
  getModelPlatforms: () => Get('/admin/models/platforms'),

  // User API Keys
  getAPIKeys: () => Get('/api-keys'),
  getAPIKey: (id) => Get(`/api-keys/${id}`),
  createAPIKey: (data) => Post('/api-keys', data),
  updateAPIKey: (id, data) => Put(`/api-keys/${id}`, data),
  deleteAPIKey: (id) => Delete(`/api-keys/${id}`),
  toggleAPIKey: (id) => Put(`/api-keys/${id}/toggle`),

  // Admin - User API Keys (管理员管理用户的 API Key)
  adminGetUserAPIKeys: (userId) => Get(`/admin/users/${userId}/api-keys`),
  adminCreateUserAPIKey: (userId, data) => Post(`/admin/users/${userId}/api-keys`, data),
  adminDeleteUserAPIKey: (userId, keyId) => Delete(`/admin/users/${userId}/api-keys/${keyId}`),
  adminToggleUserAPIKey: (userId, keyId) => Put(`/admin/users/${userId}/api-keys/${keyId}/toggle`),

  // Admin - All API Keys (管理员查看所有 API Key)
  adminGetAllAPIKeys: (params) => Get('/admin/api-keys', { params }),
  adminGetAPIKeyLogs: (keyId, params) => Get(`/admin/api-keys/${keyId}/logs`, { params }),

  // Admin - User Rate Management
  batchUpdateUserRates: (data) => Post('/admin/users/batch-rate', data),

  // Admin - User Usage Statistics
  getUserUsageStats: (id, params) => Get(`/admin/users/${id}/usage`, { params }),
  getUserUsageSummary: (id) => Get(`/admin/users/${id}/usage/summary`),
  getUserUsageStatsDB: (id, params) => Get(`/admin/users/${id}/usage/db`, { params }),

  // User Usage Statistics (for current user)
  getMyUsageStats: (params) => Get('/usage/stats', { params }),
  getMyUsageSummary: () => Get('/usage/summary'),
  getMyDailyUsage: (params) => Get('/usage/daily', { params }),
  getMyModelUsage: (params) => Get('/usage/models', { params }),

  // Admin - Cache Management
  getCacheStats: () => Get('/admin/cache/stats'),
  getCacheSessions: (params) => Get('/admin/cache/sessions', { params }),
  removeCacheSession: (sessionId) => Delete(`/admin/cache/sessions/${sessionId}`),
  getCacheAccounts: () => Get('/admin/cache/accounts'),   // 获取有缓存的账号列表（聚合）
  getCacheUsers: () => Get('/admin/cache/users'),         // 获取有缓存的用户列表（聚合）
  getUnavailableAccounts: () => Get('/admin/cache/unavailable'),
  clearCache: (type) => Post('/admin/cache/clear', { type }),
  clearUserCache: (userId) => Delete(`/admin/cache/users/${userId}`),
  clearAPIKeyCache: (apiKeyId) => Delete(`/admin/cache/api-keys/${apiKeyId}`),

  // Admin - Account Cache
  clearAccountSessions: (accountId) => Delete(`/admin/accounts/${accountId}/cache/sessions`),
  markAccountUnavailable: (accountId, data) => Post(`/admin/accounts/${accountId}/cache/unavailable`, data),
  clearAccountUnavailable: (accountId) => Delete(`/admin/accounts/${accountId}/cache/unavailable`),
  getAccountConcurrency: (accountId) => Get(`/admin/accounts/${accountId}/cache/concurrency`),
  setAccountConcurrencyLimit: (accountId, limit) => Put(`/admin/accounts/${accountId}/cache/concurrency`, { limit }),
  resetAccountConcurrency: (accountId) => Delete(`/admin/accounts/${accountId}/cache/concurrency`),

  // Admin - Cache Config
  getCacheConfig: () => Get('/admin/cache/config'),
  updateCacheConfig: (data) => Put('/admin/cache/config', data),

  // Admin - User Concurrency
  getUserConcurrency: (userId) => Get(`/admin/users/${userId}/concurrency`),
  resetUserConcurrency: (userId) => Delete(`/admin/users/${userId}/concurrency`),

  // Admin - System Configs
  getSystemConfigs: () => Get('/admin/configs'),
  getSystemConfigsByCategory: (category) => Get(`/admin/configs/category/${category}`),
  updateSystemConfigs: (configs) => Put('/admin/configs', { configs }),
  getSyncStatus: () => Get('/admin/configs/sync/status'),
  triggerSync: () => Post('/admin/configs/sync/trigger'),

  // Admin - All Users (no pagination)
  getAllUsers: () => Get('/admin/users/all'),
  batchUpdatePriceRate: (data) => Post('/admin/users/batch-price-rate', data),
  updateAllPriceRate: (data) => Post('/admin/users/all-price-rate', data),

  // Admin - Packages (套餐模板管理)
  getPackages: () => Get('/admin/packages'),
  createPackage: (data) => Post('/admin/packages', data),
  updatePackage: (id, data) => Put(`/admin/packages/${id}`, data),
  deletePackage: (id) => Delete(`/admin/packages/${id}`),

  // Admin - User Packages (用户套餐管理)
  getUserPackages: (userId) => Get(`/admin/user-packages/user/${userId}`),
  assignUserPackage: (userId, data) => Post(`/admin/user-packages/user/${userId}`, data),
  updateUserPackage: (id, data) => Put(`/admin/user-packages/${id}`, data),
  deleteUserPackage: (id) => Delete(`/admin/user-packages/${id}`),

  // User Packages (用户自己的套餐)
  getMyPackages: () => Get('/my-packages'),
  getMyActivePackages: () => Get('/my-packages/active'),
  getAvailablePackages: () => Get('/packages'),

  // Admin - Proxy Configs (代理配置管理)
  getProxyConfigs: (params) => Get('/admin/proxy-configs', { params }),
  getEnabledProxyConfigs: () => Get('/admin/proxy-configs/enabled'),
  getDefaultProxyConfig: () => Get('/admin/proxy-configs/default'),
  setDefaultProxyConfig: (id) => Put(`/admin/proxy-configs/${id}/default`),
  clearDefaultProxyConfig: () => Delete('/admin/proxy-configs/default'),
  getProxyConfig: (id) => Get(`/admin/proxy-configs/${id}`),
  createProxyConfig: (data) => Post('/admin/proxy-configs', data),
  updateProxyConfig: (id, data) => Put(`/admin/proxy-configs/${id}`, data),
  deleteProxyConfig: (id) => Delete(`/admin/proxy-configs/${id}`),
  toggleProxyConfig: (id) => Put(`/admin/proxy-configs/${id}/toggle`),
  testProxyConnectivity: (data) => Post('/admin/proxy-configs/test', data),

  // Admin - Operation Logs (操作日志)
  getOperationLogs: (params) => Get('/admin/operation-logs', { params }),
  getOperationLog: (id) => Get(`/admin/operation-logs/${id}`),
  getOperationLogStats: (params) => Get('/admin/operation-logs/stats', { params }),
  cleanupOperationLogs: (days) => Delete(`/admin/operation-logs/cleanup?days=${days}`),

  // Admin - Client Filter (客户端过滤)
  getClientFilterConfig: () => Get('/admin/client-filter/config'),
  updateClientFilterConfig: (data) => Put('/admin/client-filter/config', data),
  reloadClientFilterCache: () => Post('/admin/client-filter/reload'),
  testClientFilter: (data) => Post('/admin/client-filter/test', data),

  // Admin - Client Types (客户端类型)
  getClientTypes: () => Get('/admin/client-filter/client-types'),
  getClientType: (id) => Get(`/admin/client-filter/client-types/${id}`),
  createClientType: (data) => Post('/admin/client-filter/client-types', data),
  updateClientType: (id, data) => Put(`/admin/client-filter/client-types/${id}`, data),
  deleteClientType: (id) => Delete(`/admin/client-filter/client-types/${id}`),
  toggleClientType: (id) => Put(`/admin/client-filter/client-types/${id}/toggle`),

  // Admin - Filter Rules (过滤规则)
  getFilterRules: (params) => Get('/admin/client-filter/rules', { params }),
  getFilterRule: (id) => Get(`/admin/client-filter/rules/${id}`),
  createFilterRule: (data) => Post('/admin/client-filter/rules', data),
  updateFilterRule: (id, data) => Put(`/admin/client-filter/rules/${id}`, data),
  deleteFilterRule: (id) => Delete(`/admin/client-filter/rules/${id}`),
  toggleFilterRule: (id) => Put(`/admin/client-filter/rules/${id}/toggle`),

  // Admin - System Monitor (系统监控)
  getMonitorData: () => Get('/admin/monitor'),
  getSystemStats: () => Get('/admin/monitor/system'),
  getRedisStats: () => Get('/admin/monitor/redis'),
  getMySQLStats: () => Get('/admin/monitor/mysql'),
  getAccountStats: () => Get('/admin/monitor/accounts'),
  getUserStats: () => Get('/admin/monitor/users'),
  getTodayUsageStats: () => Get('/admin/monitor/today'),

  // Admin - System Logs (系统日志)
  getSystemLogFiles: (params) => Get('/admin/system-logs/files', { params }),
  readSystemLog: (params) => Get('/admin/system-logs/read', { params }),
  tailSystemLog: (params) => Get('/admin/system-logs/tail', { params }),
  downloadSystemLog: (file, source = 'app') => `/api/admin/system-logs/download?file=${encodeURIComponent(file)}&source=${source}`,
  deleteSystemLog: (file, source = 'app') => Delete(`/admin/system-logs/file?file=${encodeURIComponent(file)}&source=${source}`),

  // Admin - Error Messages (错误消息配置)
  getErrorMessages: () => Get('/admin/error-messages'),
  getErrorMessage: (id) => Get(`/admin/error-messages/${id}`),
  getErrorMessagesByCode: (code) => Get(`/admin/error-messages/code/${code}`),
  createErrorMessage: (data) => Post('/admin/error-messages', data),
  updateErrorMessage: (id, data) => Put(`/admin/error-messages/${id}`, data),
  deleteErrorMessage: (id) => Delete(`/admin/error-messages/${id}`),
  toggleErrorMessage: (id) => Put(`/admin/error-messages/${id}/toggle`),
  initErrorMessages: () => Post('/admin/error-messages/init'),
  refreshErrorMessages: () => Post('/admin/error-messages/refresh'),
  enableAllErrorMessages: () => Put('/admin/error-messages/enable-all'),
  disableAllErrorMessages: () => Put('/admin/error-messages/disable-all')
}
