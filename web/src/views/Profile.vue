<!--
 * 文件作用：个人设置页面，用户修改个人信息
 * 负责功能：
 *   - 基本信息修改（邮箱）
 *   - 密码修改
 *   - 用户配置保存
 * 重要程度：⭐⭐ 辅助（用户配置）
 * 依赖模块：element-plus, user store, api
-->
<template>
  <div class="profile-page">
    <h2>个人设置</h2>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card header="基本信息">
          <el-form :model="profileForm" label-width="80px">
            <el-form-item label="用户名">
              <el-input :value="userStore.user?.username" disabled />
            </el-form-item>
            <el-form-item label="邮箱">
              <el-input v-model="profileForm.email" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingProfile" @click="handleSaveProfile">
                保存
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card header="修改密码">
          <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="80px">
            <el-form-item label="原密码" prop="old_password">
              <el-input v-model="pwdForm.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input v-model="pwdForm.new_password" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingPwd" @click="handleChangePassword">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import api from '@/api'

const userStore = useUserStore()

const profileForm = reactive({ email: '' })
const savingProfile = ref(false)

const pwdFormRef = ref()
const pwdForm = reactive({ old_password: '', new_password: '' })
const savingPwd = ref(false)

const pwdRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' }
  ]
}

onMounted(() => {
  profileForm.email = userStore.user?.email || ''
})

async function handleSaveProfile() {
  savingProfile.value = true
  try {
    await api.updateProfile({ email: profileForm.email })
    await userStore.fetchProfile()
    ElMessage.success('保存成功')
  } catch (e) {
    // handled
  } finally {
    savingProfile.value = false
  }
}

async function handleChangePassword() {
  const valid = await pwdFormRef.value.validate().catch(() => false)
  if (!valid) return

  savingPwd.value = true
  try {
    await api.changePassword(pwdForm)
    ElMessage.success('密码修改成功')
    pwdForm.old_password = ''
    pwdForm.new_password = ''
  } catch (e) {
    // handled
  } finally {
    savingPwd.value = false
  }
}
</script>

<style scoped>
.profile-page h2 {
  margin-bottom: 20px;
  color: #333;
}
</style>
