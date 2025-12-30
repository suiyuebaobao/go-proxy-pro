<template>
  <div class="dashboard">
    <h2>仪表盘</h2>
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>API 请求</template>
          <div class="stat-value">--</div>
          <div class="stat-label">今日请求数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>账户</template>
          <div class="stat-value">--</div>
          <div class="stat-label">活跃账户数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>费用</template>
          <div class="stat-value">$--</div>
          <div class="stat-label">今日费用</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>用户</template>
          <div class="stat-value">--</div>
          <div class="stat-label">活跃用户数</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="welcome-card">
      <h3>欢迎使用 Go-AIProxy</h3>
      <p>统一管理多个 AI 平台账户，提供安全的 API 代理服务。</p>
      <el-alert
        title="M1 基础框架已完成"
        type="success"
        description="用户管理、JWT 认证、基础 CRUD 功能已就绪。下一步将开发 M2 账户管理模块。"
        show-icon
        :closable="false"
      />
    </el-card>
  </div>
</template>

<script>
export default {
  name: 'Dashboard'
}
</script>

<style scoped>
.dashboard h2 {
  margin-bottom: 20px;
  color: #333;
}

.stat-cards {
  margin-bottom: 20px;
}

.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
}

.stat-label {
  color: #999;
  margin-top: 8px;
}

.welcome-card {
  margin-top: 20px;
}

.welcome-card h3 {
  margin-bottom: 10px;
}

.welcome-card p {
  color: #666;
  margin-bottom: 20px;
}
</style>
