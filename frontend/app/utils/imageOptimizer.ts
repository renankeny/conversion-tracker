/**
 * Otimiza imagens selecionadas pelo comprador no navegador (Logo ou Favicon),
 * mantendo SVGs intactos e redimensionando PNG/JPG/WebP via <canvas> para um
 * DataURL leve (~10KB a 50KB), pronto para salvar diretamente no PostgreSQL.
 */
export async function optimizeBrandingImage(
  file: File,
  kind: 'logo' | 'favicon'
): Promise<string> {
  const maxBytes = 2 * 1024 * 1024 // 2 MB de entrada
  if (file.size > maxBytes) {
    throw new Error('O arquivo selecionado excede 2MB. Escolha uma imagem menor.')
  }

  // SVGs ou ícones .ico pequenos já estão otimizados
  if (file.type === 'image/svg+xml' || (kind === 'favicon' && file.size <= 32 * 1024)) {
    return readFileAsDataURL(file)
  }

  const rawDataUrl = await readFileAsDataURL(file)
  const img = await loadImageElement(rawDataUrl)

  const maxWidth = kind === 'logo' ? 420 : 64
  const maxHeight = kind === 'logo' ? 140 : 64

  let width = img.naturalWidth || maxWidth
  let height = img.naturalHeight || maxHeight

  if (width > maxWidth || height > maxHeight) {
    const ratio = Math.min(maxWidth / width, maxHeight / height)
    width = Math.max(1, Math.round(width * ratio))
    height = Math.max(1, Math.round(height * ratio))
  }

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    return rawDataUrl
  }

  ctx.clearRect(0, 0, width, height)
  ctx.drawImage(img, 0, 0, width, height)

  // Tenta exportar em WebP com transparência (mais leve); fallback para PNG
  const webp = canvas.toDataURL('image/webp', 0.9)
  if (webp.startsWith('data:image/webp')) {
    return webp
  }
  return canvas.toDataURL('image/png')
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('Falha ao ler o arquivo de imagem'))
    reader.readAsDataURL(file)
  })
}

function loadImageElement(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('Formato de imagem inválido'))
    img.src = src
  })
}

