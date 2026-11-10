export const EVENT_TYPES = {
  PAGE_VIEW: 'PageView',
  VIEW_CONTENT: 'ViewContent',
  ADD_TO_CART: 'AddToCart',
  INITIATE_CHECKOUT: 'InitiateCheckout',
  PURCHASE: 'Purchase',
  LEAD: 'Lead',
  COMPLETE_REGISTRATION: 'CompleteRegistration',
  SEARCH: 'Search',
  ADD_PAYMENT_INFO: 'AddPaymentInfo',
  ADD_TO_WISHLIST: 'AddToWishlist',
} as const

export type EventType = typeof EVENT_TYPES[keyof typeof EVENT_TYPES]

export const EVENT_TYPE_LABELS: Record<string, string> = {
  PageView: 'Visualização de Página',
  ViewContent: 'Visualização de Conteúdo',
  AddToCart: 'Adicionar ao Carrinho',
  InitiateCheckout: 'Iniciar Checkout',
  Purchase: 'Compra',
  Lead: 'Lead',
  CompleteRegistration: 'Registro Completo',
  Search: 'Busca',
  AddPaymentInfo: 'Adicionar Info de Pagamento',
  AddToWishlist: 'Adicionar à Lista de Desejos',
}
