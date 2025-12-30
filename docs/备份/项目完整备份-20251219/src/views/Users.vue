<template>
  <div class="users-page">
    <div class="page-header">
      <h2>用户管理</h2>
      <div class="header-actions">
        <el-button type="primary" @click="showCreateUserDialog">
          <el-icon><Plus /></el-icon> 创建用户
        </el-button>
        <el-button :disabled="selectedUsers.length === 0" @click="showBatchRateDialog">
          批量设置倍率 ({{ selectedUsers.length }})
        </el-button>
      </div>
    </div>

    <!-- 用户列表 -->
    <el-card>
      <el-table :data="users" v-loading="loading" stripe @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="55" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'info'">
              {{ row.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'warning'">
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="price_rate" label="价格倍率" width="100">
          <template #default="{ row }">
            <el-tag :type="getPriceRateType(row.price_rate)">
              {{ row.price_rate === 0 ? '免费' : `${row.price_rate}x` }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="max_concurrency" label="并发限制" width="90">
          <template #default="{ row }">
            {{ row.max_concurrency || 10 }}
          </template>
        </el-table-column>
        <el-table-column prop="quota_key_balance" label="额度余额" width="100">
          <template #default="{ row }">
            <span :class="{ 'low-balance': row.quota_key_balance < 1 && row.quota_key_balance > 0 }">
              ${{ (row.quota_key_balance || 0).toFixed(2) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="subscription_daily_remain" label="订阅日余额" width="110">
          <template #default="{ row }">
            <span v-if="row.subscription_daily_remain === -1" class="unlimited">无限</span>
            <span v-else :class="{ 'low-balance': row.subscription_daily_remain < 1 && row.subscription_daily_remain > 0 }">
              ${{ (row.subscription_daily_remain || 0).toFixed(2) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="320" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button type="success" link @click="viewAPIKeys(row)">apikey</el-button>
            <el-button type="warning" link @click="viewPackages(row)">套餐</el-button>
            <el-button type="info" link @click="viewUsage(row)">统计</el-button>
            <el-popconfirm
              title="确定删除该用户吗？"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button type="danger" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @change="fetchUsers"
        />
      </div>
    </el-card>

    <!-- 编辑弹窗 -->
    <el-dialog v-model="dialogVisible" title="编辑用户" width="500">
      <el-form ref="formRef" :model="editForm" :rules="rules" label-width="80px">
        <el-form-item label="用户名">
          <el-input :value="editForm.username" disabled />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="editForm.role" style="width: 100%">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="editForm.status" style="width: 100%">
            <el-option label="正常" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格倍率" prop="price_rate">
          <el-input-number v-model="editForm.price_rate" :min="0" :max="10" :step="0.1" :precision="2" style="width: 100%" />
          <div class="form-tip">0 = 免费，1 = 原价，1.5 = 1.5倍</div>
        </el-form-item>
        <el-form-item label="最大并发">
          <el-input-number v-model="editForm.max_concurrency" :min="1" :max="100" style="width: 100%" />
          <div class="form-tip">用户同时进行的最大请求数</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 创建用户弹窗 -->
    <el-dialog v-model="createUserDialogVisible" title="创建用户" width="500" :close-on-click-modal="false">
      <el-form ref="createUserFormRef" :model="createUserForm" :rules="createUserRules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="createUserForm.username" placeholder="请输入用户名" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="createUserForm.password" type="password" placeholder="请输入密码" show-password />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="createUserForm.email" placeholder="可选" />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="createUserForm.role" style="width: 100%">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-select v-model="createUserForm.status" style="width: 100%">
            <el-option label="正常" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格倍率" prop="price_rate">
          <el-input-number v-model="createUserForm.price_rate" :min="0" :max="10" :step="0.1" :precision="2" style="width: 100%" />
          <div class="form-tip">0 = 免费，1 = 原价，1.5 = 1.5倍</div>
        </el-form-item>
        <el-form-item label="最大并发">
          <el-input-number v-model="createUserForm.max_concurrency" :min="1" :max="100" style="width: 100%" />
          <div class="form-tip">用户同时进行的最大请求数</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createUserDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingUser" @click="handleCreateUser">创建</el-button>
      </template>
    </el-dialog>

    <!-- 批量设置倍率弹窗 -->
    <el-dialog v-model="batchRateDialogVisible" title="批量设置价格倍率" width="450">
      <el-form label-width="100px">
        <el-form-item label="选中用户">
          <div class="selected-users">
            <el-tag v-for="user in selectedUsers" :key="user.id" style="margin: 2px;">
              {{ user.username }}
            </el-tag>
          </div>
        </el-form-item>
        <el-form-item label="价格倍率">
          <el-input-number v-model="batchRate" :min="0" :max="10" :step="0.1" :precision="2" style="width: 100%" />
          <div class="form-tip">0 = 免费，1 = 原价，1.5 = 1.5倍</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchRateDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="batchSubmitting" @click="submitBatchRate">确认设置</el-button>
      </template>
    </el-dialog>

    <!-- 用户使用统计弹窗 -->
    <el-dialog v-model="usageDialogVisible" :title="`${usageUser?.username} 的使用统计`" width="800">
      <div v-loading="usageLoading">
        <!-- 统计概览 -->
        <el-row :gutter="16" class="usage-summary">
          <el-col :span="8">
            <el-statistic title="总请求数" :value="usageSummary.totalRequests" />
          </el-col>
          <el-col :span="8">
            <el-statistic title="总 Token" :value="usageSummary.totalTokens" />
          </el-col>
          <el-col :span="8">
            <el-statistic title="总消费 ($)" :value="usageSummary.totalCost" :precision="4" />
          </el-col>
        </el-row>

        <!-- 套餐使用情况 -->
        <div class="usage-section" v-if="packageUsage.length > 0">
          <h4>套餐使用情况</h4>
          <el-table :data="packageUsage" size="small" max-height="250">
            <el-table-column prop="name" label="套餐名称" width="120" />
            <el-table-column prop="type" label="类型" width="80">
              <template #default="{ row }">
                <el-tag :type="row.type === 'subscription' ? 'primary' : 'success'" size="small">
                  {{ row.type === 'subscription' ? '订阅' : '额度' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                  {{ row.status === 'active' ? '有效' : row.status === 'expired' ? '已过期' : '已用尽' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="使用情况" min-width="200">
              <template #default="{ row }">
                <template v-if="row.type === 'subscription'">
                  <div class="pkg-usage-info">
                    <span v-if="row.daily_quota > 0">日: ${{ row.daily_used?.toFixed(2) || '0.00' }}/${{ row.daily_quota }}</span>
                    <span v-if="row.weekly_quota > 0">周: ${{ row.weekly_used?.toFixed(2) || '0.00' }}/${{ row.weekly_quota }}</span>
                    <span v-if="row.monthly_quota > 0">月: ${{ row.monthly_used?.toFixed(2) || '0.00' }}/${{ row.monthly_quota }}</span>
                    <span v-if="!row.daily_quota && !row.weekly_quota && !row.monthly_quota" class="unlimited">无限额</span>
                  </div>
                </template>
                <template v-else>
                  <div>
                    <span>${{ row.quota_used?.toFixed(2) || '0.00' }} / ${{ row.quota_total?.toFixed(2) || '0.00' }}</span>
                    <el-progress
                      :percentage="row.quota_total > 0 ? Math.min(100, (row.quota_used / row.quota_total) * 100) : 0"
                      :status="row.quota_used >= row.quota_total ? 'exception' : ''"
                      :stroke-width="4"
                      style="margin-top: 4px"
                    />
                  </div>
                </template>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <el-empty v-else-if="!usageLoading" description="暂无套餐" :image-size="60" />

        <!-- 按模型使用统计 -->
        <div class="usage-section">
          <h4>按模型统计</h4>
          <el-table :data="modelUsage" size="small" max-height="200">
            <el-table-column prop="model" label="模型" />
            <el-table-column prop="requests" label="请求数" width="100" />
            <el-table-column prop="tokens" label="Token 数" width="120" />
            <el-table-column prop="cost" label="费用 ($)" width="100">
              <template #default="{ row }">{{ row.cost.toFixed(4) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="usageDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- API Key 管理弹窗 -->
    <el-dialog v-model="apiKeyDialogVisible" :title="`${apiKeyUser?.username} 的 apikey`" width="900">
      <div class="apikey-header">
        <el-button type="primary" size="small" @click="showCreateAPIKeyDialog">
          <el-icon><Plus /></el-icon> 创建 apikey
        </el-button>
      </div>
      <el-table :data="apiKeys" v-loading="apiKeyLoading" stripe size="small">
        <el-table-column prop="name" label="名称" width="100" />
        <el-table-column label="Key" min-width="280">
          <template #default="{ row }">
            <code class="key-full">{{ row.key_full || row.key_prefix }}</code>
          </template>
        </el-table-column>
        <el-table-column label="绑定套餐" min-width="120">
          <template #default="{ row }">
            <div v-if="row.user_package">
              <div>{{ row.user_package.name }}</div>
              <el-tag :type="row.billing_type === 'subscription' ? 'primary' : 'success'" size="small">
                {{ row.billing_type === 'subscription' ? '订阅' : '额度' }}
              </el-tag>
            </div>
            <el-tag v-else type="info" size="small">未绑定</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="70">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="rate_limit" label="限制" width="80">
          <template #default="{ row }">
            {{ row.rate_limit }}/分钟
          </template>
        </el-table-column>
        <el-table-column prop="request_count" label="请求数" width="70" />
        <el-table-column label="费用" width="90">
          <template #default="{ row }">
            ${{ (row.cost_used || 0).toFixed(4) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link :type="row.status === 'active' ? 'warning' : 'success'" size="small" @click="handleToggleAPIKey(row)">
              {{ row.status === 'active' ? '禁用' : '启用' }}
            </el-button>
            <el-popconfirm title="确定删除此 apikey？" @confirm="handleDeleteAPIKey(row)">
              <template #reference>
                <el-button link type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!apiKeyLoading && apiKeys.length === 0" description="暂无 apikey" />
      <template #footer>
        <el-button @click="apiKeyDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 创建 API Key 弹窗 -->
    <el-dialog v-model="createKeyDialogVisible" title="创建 apikey" width="500" :close-on-click-modal="false">
      <el-form :model="createKeyForm" :rules="createKeyRules" ref="createKeyFormRef" label-position="top">
        <el-form-item label="名称" prop="name">
          <el-input v-model="createKeyForm.name" placeholder="为这个 Key 起个名字" />
        </el-form-item>
        <el-form-item label="绑定套餐" prop="user_package_id">
          <el-select v-model="createKeyForm.user_package_id" style="width: 100%" placeholder="请选择要绑定的套餐">
            <el-option
              v-for="pkg in userPackagesForKey"
              :key="pkg.id"
              :label="`${pkg.name} (${pkg.type === 'subscription' ? '订阅' : '额度'})`"
              :value="pkg.id"
            />
          </el-select>
          <div class="form-tip">API Key 将从绑定的套餐扣费，一个套餐可创建多个 Key</div>
        </el-form-item>
        <el-form-item label="允许的平台">
          <el-select v-model="createKeyForm.allowed_platforms" style="width: 100%">
            <el-option label="全部平台" value="all" />
            <el-option label="仅 Claude" value="claude" />
            <el-option label="仅 OpenAI" value="openai" />
            <el-option label="仅 Gemini" value="gemini" />
          </el-select>
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="每分钟限制">
              <el-input-number v-model="createKeyForm.rate_limit" :min="1" :max="1000" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="每日限制 (0=不限)">
              <el-input-number v-model="createKeyForm.daily_limit" :min="0" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createKeyDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingKey" @click="handleCreateAPIKey">创建</el-button>
      </template>
    </el-dialog>

    <!-- 显示新创建的 API Key -->
    <el-dialog v-model="showNewKeyDialog" title="API Key 已创建" width="500" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" style="margin-bottom: 16px">
        请立即复制并妥善保存 API Key，关闭后无法再次查看！
      </el-alert>
      <div class="new-key-display">
        <label>API Key:</label>
        <div class="key-box">
          <code>{{ newKey }}</code>
          <el-button type="primary" size="small" @click="copyKey">
            {{ copied ? '已复制' : '复制' }}
          </el-button>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="showNewKeyDialog = false">我已保存，关闭</el-button>
      </template>
    </el-dialog>

    <!-- 用户套餐管理弹窗 -->
    <el-dialog v-model="packageDialogVisible" :title="`${packageUser?.username} 的套餐`" width="800">
      <div class="package-header">
        <el-button type="primary" size="small" @click="showAssignPackageDialog">
          <el-icon><Plus /></el-icon> 分配套餐
        </el-button>
      </div>
      <el-table :data="userPackages" v-loading="packageLoading" stripe size="small">
        <el-table-column prop="name" label="套餐名称" width="120" />
        <el-table-column prop="type" label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="row.type === 'subscription' ? 'primary' : 'success'" size="small">
              {{ row.type === 'subscription' ? '订阅' : '额度' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status === 'active' ? '有效' : row.status === 'expired' ? '已过期' : '已用尽' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="详情" min-width="280">
          <template #default="{ row }">
            <template v-if="row.type === 'subscription'">
              <div class="subscription-info">
                <div class="expire-info">有效期: {{ formatDate(row.expire_time) }}</div>
                <div class="quota-usage">
                  <span v-if="row.daily_quota > 0" class="quota-item">
                    日: ${{ (row.daily_used || 0).toFixed(2) }}/${{ row.daily_quota }}
                  </span>
                  <span v-if="row.weekly_quota > 0" class="quota-item">
                    周: ${{ (row.weekly_used || 0).toFixed(2) }}/${{ row.weekly_quota }}
                  </span>
                  <span v-if="row.monthly_quota > 0" class="quota-item">
                    月: ${{ (row.monthly_used || 0).toFixed(2) }}/${{ row.monthly_quota }}
                  </span>
                  <span v-if="!row.daily_quota && !row.weekly_quota && !row.monthly_quota" class="quota-item unlimited">
                    无限额
                  </span>
                </div>
              </div>
            </template>
            <template v-else>
              <span>额度: ${{ (row.quota_used || 0).toFixed(2) }} / ${{ (row.quota_total || 0).toFixed(2) }}</span>
              <el-progress
                :percentage="row.quota_total > 0 ? Math.min(100, (row.quota_used / row.quota_total) * 100) : 0"
                :status="row.quota_used >= row.quota_total ? 'exception' : ''"
                :stroke-width="6"
                style="margin-top: 4px"
              />
            </template>
          </template>
        </el-table-column>
        <el-table-column label="模型限制" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.allowed_models" size="small" type="warning">
              {{ row.allowed_models.split(',').length }}个
            </el-tag>
            <span v-else class="text-muted">全部</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEditUserPackage(row)">编辑</el-button>
            <el-popconfirm title="确定删除此套餐？" @confirm="handleDeleteUserPackage(row.id)">
              <template #reference>
                <el-button type="danger" link size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!packageLoading && userPackages.length === 0" description="暂无套餐" />
      <template #footer>
        <el-button @click="packageDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 分配套餐弹窗 -->
    <el-dialog v-model="assignPackageDialogVisible" title="分配套餐" width="400" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="选择套餐">
          <el-select v-model="selectedPackageId" style="width: 100%" placeholder="请选择套餐">
            <el-option
              v-for="pkg in availablePackages"
              :key="pkg.id"
              :label="`${pkg.name} (${pkg.type === 'subscription' ? '订阅' : '额度'})`"
              :value="pkg.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignPackageDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="assigningPackage" @click="handleAssignPackage">分配</el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户套餐弹窗 -->
    <el-dialog v-model="editPackageDialogVisible" title="编辑用户套餐" width="550" :close-on-click-modal="false">
      <el-form :model="editPackageForm" label-width="100px">
        <el-form-item label="状态">
          <el-select v-model="editPackageForm.status" style="width: 100%">
            <el-option label="有效" value="active" />
            <el-option label="已过期" value="expired" />
            <el-option label="已用尽" value="exhausted" />
          </el-select>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="editPackageForm.expire_time"
            type="datetime"
            style="width: 100%"
            placeholder="选择过期时间"
          />
        </el-form-item>

        <!-- 订阅类型的周期额度 -->
        <template v-if="editPackageForm.type === 'subscription'">
          <el-divider content-position="left">周期额度限制 (0=不限)</el-divider>
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="日限额($)">
                <el-input-number v-model="editPackageForm.daily_quota" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="周限额($)">
                <el-input-number v-model="editPackageForm.weekly_quota" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="月限额($)">
                <el-input-number v-model="editPackageForm.monthly_quota" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-divider content-position="left">当前周期已用 (可重置)</el-divider>
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="日已用($)">
                <el-input-number v-model="editPackageForm.daily_used" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="周已用($)">
                <el-input-number v-model="editPackageForm.weekly_used" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="月已用($)">
                <el-input-number v-model="editPackageForm.monthly_used" :min="0" :precision="2" size="small" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <!-- 额度类型字段 -->
        <template v-if="editPackageForm.type === 'quota'">
          <el-form-item label="总额度($)">
            <el-input-number v-model="editPackageForm.quota_total" :min="0" :precision="2" style="width: 100%" />
          </el-form-item>
          <el-form-item label="已用额度($)">
            <el-input-number v-model="editPackageForm.quota_used" :min="0" :precision="2" style="width: 100%" />
          </el-form-item>
        </template>

        <el-form-item label="允许的模型">
          <el-select
            v-model="editPackageSelectedModels"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示全部模型"
            style="width: 100%"
          >
            <el-option
              v-for="model in modelList"
              :key="model.id"
              :label="model.name"
              :value="model.name"
            />
          </el-select>
          <div class="form-tip">限制该套餐可使用的模型，不选则允许全部模型</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editPackageDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingPackage" @click="handleSaveUserPackage">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '@/api'

const loading = ref(false)
const users = ref([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const modelList = ref([])

const dialogVisible = ref(false)
const submitting = ref(false)
const formRef = ref()
const editForm = reactive({ id: 0, username: '', email: '', role: '', status: '', price_rate: 1.0, max_concurrency: 10 })

// 批量设置倍率相关
const selectedUsers = ref([])
const batchRateDialogVisible = ref(false)
const batchRate = ref(1.0)
const batchSubmitting = ref(false)

// 用户使用统计相关
const usageDialogVisible = ref(false)
const usageLoading = ref(false)
const usageUser = ref(null)
const usageSummary = reactive({ totalRequests: 0, totalTokens: 0, totalCost: 0 })
const modelUsage = ref([])
const packageUsage = ref([])

// API Key 管理相关
const apiKeyDialogVisible = ref(false)
const apiKeyLoading = ref(false)
const apiKeyUser = ref(null)
const apiKeys = ref([])

// 创建 API Key 相关
const createKeyDialogVisible = ref(false)
const createKeyFormRef = ref()
const creatingKey = ref(false)
const userPackagesForKey = ref([])  // 用户套餐列表（用于选择绑定）
const createKeyForm = ref({
  name: '',
  user_package_id: null,
  allowed_platforms: 'all',
  rate_limit: 60,
  daily_limit: 0
})
const createKeyRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  user_package_id: [{ required: true, message: '请选择要绑定的套餐', trigger: 'change' }]
}

// 显示新 Key
const showNewKeyDialog = ref(false)
const newKey = ref('')
const copied = ref(false)

// 套餐管理相关
const packageDialogVisible = ref(false)
const packageLoading = ref(false)
const packageUser = ref(null)
const userPackages = ref([])
const availablePackages = ref([])

// 分配套餐相关
const assignPackageDialogVisible = ref(false)
const selectedPackageId = ref(null)
const assigningPackage = ref(false)

// 编辑用户套餐相关
const editPackageDialogVisible = ref(false)
const savingPackage = ref(false)
const editPackageForm = ref({
  id: 0,
  type: '',
  status: '',
  expire_time: null,
  daily_quota: 0,
  weekly_quota: 0,
  monthly_quota: 0,
  daily_used: 0,
  weekly_used: 0,
  monthly_used: 0,
  quota_total: 0,
  quota_used: 0,
  allowed_models: ''
})

// editPackageSelectedModels 是数组，和 editPackageForm.allowed_models (逗号分隔字符串) 双向转换
const editPackageSelectedModels = computed({
  get() {
    if (!editPackageForm.value.allowed_models) return []
    return editPackageForm.value.allowed_models.split(',').filter(m => m.trim())
  },
  set(val) {
    editPackageForm.value.allowed_models = val.join(',')
  }
})

const rules = {
  email: [{ type: 'email', message: '请输入有效的邮箱', trigger: 'blur' }]
}

// 创建用户相关
const createUserDialogVisible = ref(false)
const createUserFormRef = ref()
const creatingUser = ref(false)
const createUserForm = ref({
  username: '',
  password: '',
  email: '',
  role: 'user',
  status: 'active',
  price_rate: 1.0,
  max_concurrency: 10
})
const createUserRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名长度 3-50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 个字符', trigger: 'blur' }
  ],
  email: [{ type: 'email', message: '请输入有效的邮箱', trigger: 'blur' }]
}

function getPriceRateType(rate) {
  if (rate === 0) return 'success'
  if (rate < 1) return 'warning'
  if (rate > 1) return 'danger'
  return 'info'
}

function formatDate(str) {
  if (!str) return ''
  return new Date(str).toLocaleString('zh-CN')
}

async function fetchUsers() {
  loading.value = true
  try {
    const res = await api.getUsers({ page: pagination.page, page_size: pagination.pageSize })
    users.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (e) {
    // handled
  } finally {
    loading.value = false
  }
}

async function fetchModels() {
  try {
    const res = await api.getModels()
    modelList.value = (res.data || []).filter(m => m.enabled)
  } catch (e) {
    // handled
  }
}

function handleEdit(row) {
  Object.assign(editForm, row)
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await api.updateUser(editForm.id, {
      email: editForm.email,
      role: editForm.role,
      status: editForm.status,
      price_rate: editForm.price_rate,
      max_concurrency: editForm.max_concurrency
    })
    ElMessage.success('更新成功')
    dialogVisible.value = false
    fetchUsers()
  } catch (e) {
    // handled
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id) {
  try {
    await api.deleteUser(id)
    ElMessage.success('删除成功')
    fetchUsers()
  } catch (e) {
    // handled
  }
}

// 选择用户
function handleSelectionChange(selection) {
  selectedUsers.value = selection
}

// 显示批量设置倍率弹窗
function showBatchRateDialog() {
  batchRate.value = 1.0
  batchRateDialogVisible.value = true
}

// 提交批量设置倍率
async function submitBatchRate() {
  if (selectedUsers.value.length === 0) return

  batchSubmitting.value = true
  try {
    const userIds = selectedUsers.value.map(u => u.id)
    await api.batchUpdateUserRates({
      user_ids: userIds,
      price_rate: batchRate.value
    })
    ElMessage.success(`已更新 ${userIds.length} 个用户的价格倍率`)
    batchRateDialogVisible.value = false
    fetchUsers()
  } catch (e) {
    // handled
  } finally {
    batchSubmitting.value = false
  }
}

// 查看用户使用统计
async function viewUsage(row) {
  usageUser.value = row
  usageDialogVisible.value = true
  usageLoading.value = true

  // 重置数据
  usageSummary.totalRequests = 0
  usageSummary.totalTokens = 0
  usageSummary.totalCost = 0
  modelUsage.value = []
  packageUsage.value = []

  try {
    const res = await api.getUserUsageSummary(row.id)
    if (res.data) {
      usageSummary.totalRequests = res.data.total_requests || 0
      usageSummary.totalTokens = res.data.total_tokens || 0
      usageSummary.totalCost = res.data.total_cost || 0
      modelUsage.value = res.data.model_usage || []
      packageUsage.value = res.data.package_usage || []
    }
  } catch (e) {
    // handled
  } finally {
    usageLoading.value = false
  }
}

onMounted(() => {
  fetchUsers()
  fetchModels()
})

// ========== API Key 管理 ==========

// 查看用户的 API Key
async function viewAPIKeys(row) {
  apiKeyUser.value = row
  apiKeyDialogVisible.value = true
  apiKeyLoading.value = true
  apiKeys.value = []

  try {
    const res = await api.adminGetUserAPIKeys(row.id)
    apiKeys.value = res.data || []
  } catch (e) {
    // handled
  } finally {
    apiKeyLoading.value = false
  }
}

// 显示创建 API Key 弹窗
async function showCreateAPIKeyDialog() {
  createKeyForm.value = {
    name: '',
    user_package_id: null,
    allowed_platforms: 'all',
    rate_limit: 60,
    daily_limit: 0
  }
  // 加载用户的活跃套餐
  try {
    const res = await api.getUserPackages(apiKeyUser.value.id)
    userPackagesForKey.value = (res.data || []).filter(p => p.status === 'active')
    if (userPackagesForKey.value.length === 0) {
      ElMessage.warning('该用户没有活跃的套餐，请先分配套餐')
      return
    }
  } catch (e) {
    ElMessage.error('获取用户套餐失败')
    return
  }
  createKeyDialogVisible.value = true
}

// 创建 API Key
async function handleCreateAPIKey() {
  const valid = await createKeyFormRef.value?.validate().catch(() => false)
  if (!valid) return

  creatingKey.value = true
  try {
    const res = await api.adminCreateUserAPIKey(apiKeyUser.value.id, createKeyForm.value)
    newKey.value = res.data.key
    createKeyDialogVisible.value = false
    showNewKeyDialog.value = true
    // 刷新列表
    viewAPIKeys(apiKeyUser.value)
  } catch (e) {
    // handled
  } finally {
    creatingKey.value = false
  }
}

// 切换 API Key 状态
async function handleToggleAPIKey(row) {
  try {
    await api.adminToggleUserAPIKey(apiKeyUser.value.id, row.id)
    ElMessage.success('状态已更新')
    viewAPIKeys(apiKeyUser.value)
  } catch (e) {
    // handled
  }
}

// 删除 API Key
async function handleDeleteAPIKey(row) {
  try {
    await api.adminDeleteUserAPIKey(apiKeyUser.value.id, row.id)
    ElMessage.success('删除成功')
    viewAPIKeys(apiKeyUser.value)
  } catch (e) {
    // handled
  }
}

// 复制 Key
async function copyKey() {
  try {
    await navigator.clipboard.writeText(newKey.value)
    copied.value = true
    ElMessage.success('已复制到剪贴板')
    setTimeout(() => { copied.value = false }, 2000)
  } catch (e) {
    // 降级方案
    const input = document.createElement('input')
    input.value = newKey.value
    document.body.appendChild(input)
    input.select()
    document.execCommand('copy')
    document.body.removeChild(input)
    copied.value = true
    ElMessage.success('已复制到剪贴板')
    setTimeout(() => { copied.value = false }, 2000)
  }
}

// ========== 创建用户 ==========

// 显示创建用户弹窗
function showCreateUserDialog() {
  createUserForm.value = {
    username: '',
    password: '',
    email: '',
    role: 'user',
    status: 'active',
    price_rate: 1.0,
    max_concurrency: 10
  }
  createUserDialogVisible.value = true
}

// 创建用户
async function handleCreateUser() {
  const valid = await createUserFormRef.value?.validate().catch(() => false)
  if (!valid) return

  creatingUser.value = true
  try {
    await api.createUser(createUserForm.value)
    ElMessage.success('创建成功')
    createUserDialogVisible.value = false
    fetchUsers()
  } catch (e) {
    // handled by api
  } finally {
    creatingUser.value = false
  }
}

// ========== 套餐管理 ==========

// 查看用户套餐
async function viewPackages(row) {
  packageUser.value = row
  packageDialogVisible.value = true
  packageLoading.value = true
  userPackages.value = []

  try {
    const res = await api.getUserPackages(row.id)
    userPackages.value = res.data || []
  } catch (e) {
    // handled
  } finally {
    packageLoading.value = false
  }
}

// 显示分配套餐弹窗
async function showAssignPackageDialog() {
  selectedPackageId.value = null
  // 获取可用套餐列表
  try {
    const res = await api.getPackages()
    availablePackages.value = (res.data || []).filter(p => p.status === 'active')
  } catch (e) {
    // handled
  }
  assignPackageDialogVisible.value = true
}

// 分配套餐
async function handleAssignPackage() {
  if (!selectedPackageId.value) {
    ElMessage.warning('请选择套餐')
    return
  }

  assigningPackage.value = true
  try {
    await api.assignUserPackage(packageUser.value.id, { package_id: selectedPackageId.value })
    ElMessage.success('分配成功')
    assignPackageDialogVisible.value = false
    viewPackages(packageUser.value)
  } catch (e) {
    // handled
  } finally {
    assigningPackage.value = false
  }
}

// 编辑用户套餐
function handleEditUserPackage(row) {
  editPackageForm.value = {
    id: row.id,
    type: row.type,
    status: row.status,
    expire_time: row.expire_time,
    daily_quota: row.daily_quota || 0,
    weekly_quota: row.weekly_quota || 0,
    monthly_quota: row.monthly_quota || 0,
    daily_used: row.daily_used || 0,
    weekly_used: row.weekly_used || 0,
    monthly_used: row.monthly_used || 0,
    quota_total: row.quota_total || 0,
    quota_used: row.quota_used || 0,
    allowed_models: row.allowed_models || ''
  }
  editPackageDialogVisible.value = true
}

// 保存用户套餐
async function handleSaveUserPackage() {
  savingPackage.value = true
  try {
    const data = {
      status: editPackageForm.value.status,
      expire_time: editPackageForm.value.expire_time,
      allowed_models: editPackageForm.value.allowed_models
    }

    if (editPackageForm.value.type === 'subscription') {
      data.daily_quota = editPackageForm.value.daily_quota
      data.weekly_quota = editPackageForm.value.weekly_quota
      data.monthly_quota = editPackageForm.value.monthly_quota
      data.daily_used = editPackageForm.value.daily_used
      data.weekly_used = editPackageForm.value.weekly_used
      data.monthly_used = editPackageForm.value.monthly_used
    } else {
      data.quota_total = editPackageForm.value.quota_total
      data.quota_used = editPackageForm.value.quota_used
    }

    await api.updateUserPackage(editPackageForm.value.id, data)
    ElMessage.success('保存成功')
    editPackageDialogVisible.value = false
    viewPackages(packageUser.value)
  } catch (e) {
    // handled
  } finally {
    savingPackage.value = false
  }
}

// 删除用户套餐
async function handleDeleteUserPackage(id) {
  try {
    await api.deleteUserPackage(id)
    ElMessage.success('删除成功')
    viewPackages(packageUser.value)
  } catch (e) {
    // handled
  }
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  color: #333;
  margin: 0;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.selected-users {
  max-height: 100px;
  overflow-y: auto;
}

.usage-summary {
  margin-bottom: 24px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.usage-section {
  margin-top: 16px;
}

.usage-section h4 {
  margin: 0 0 12px 0;
  color: #333;
  font-size: 14px;
}

.pkg-usage-info {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  font-size: 12px;
}

.pkg-usage-info span {
  background: #f0f2f5;
  padding: 2px 8px;
  border-radius: 4px;
}

/* API Key 相关样式 */
.apikey-header {
  margin-bottom: 16px;
}

.key-full {
  background: #f3f4f6;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #4b5563;
  word-break: break-all;
}

.new-key-display {
  margin-top: 16px;
}

.new-key-display label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 8px;
}

.key-box {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 16px;
}

.key-box code {
  flex: 1;
  font-size: 13px;
  color: #1f2937;
  word-break: break-all;
}

.low-balance {
  color: #f56c6c;
  font-weight: bold;
}

/* 套餐管理相关样式 */
.package-header {
  margin-bottom: 16px;
}

.subscription-info {
  font-size: 12px;
}

.subscription-info .expire-info {
  margin-bottom: 4px;
}

.subscription-info .quota-usage {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.subscription-info .quota-item {
  background: #f0f2f5;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
}

.subscription-info .quota-item.unlimited {
  background: #e8f5e9;
  color: #2e7d32;
}

.text-muted {
  color: #909399;
}

.unlimited {
  color: #67c23a;
  font-weight: 500;
}
</style>
