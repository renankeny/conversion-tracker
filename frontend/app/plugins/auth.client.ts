// Plugin client-only que valida o token contra o backend ao iniciar a app.
// Roda uma única vez no carregamento da página, antes da hidratação completa.
// Se o token for inválido/expirado, limpa o estado e o middleware redireciona.
export default defineNuxtPlugin(async () => {
  const { token, user, fetchMe } = useAuth()

  // Sem token: nada a fazer.
  if (!token.value) return

  // Já temos user em cookie e token válido: só revalida em background sem bloquear.
  if (user.value) {
    fetchMe().catch(() => {})
    return
  }

  // Token sem user (caso raro): valida sincronamente pra garantir antes do middleware.
  await fetchMe()
})
