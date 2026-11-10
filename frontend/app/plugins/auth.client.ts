export default defineNuxtPlugin({
  name: 'auth-init',
  dependsOn: ['pinia', 'api'],
  async setup() {
    const { token, user, fetchMe } = useAuth()
    const { checkSetupStatus, serviceNotice } = useSetup()

    await checkSetupStatus(true)
    if (serviceNotice.value?.active) return

    if (!token.value) return

    if (user.value) {
      fetchMe().catch(() => {})
      return
    }

    await fetchMe()
  },
})
