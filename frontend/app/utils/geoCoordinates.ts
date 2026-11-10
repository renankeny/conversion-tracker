export interface GeoStat {
  country?: string
  state?: string
  city?: string
  count?: number
  event_name?: string
  [key: string]: any
}

export interface RegionFocus {
  id: string
  label: string
  targetLon: number
  targetLat: number
}

export const cityCoords: Record<string, { lat: number; lon: number }> = {
  // Japão
  tokyo: { lat: 35.6895, lon: 139.6917 }, nagoya: { lat: 35.1815, lon: 136.9066 },
  osaka: { lat: 34.6937, lon: 135.5023 }, hamamatsu: { lat: 34.7108, lon: 137.7261 },
  okazaki: { lat: 34.9553, lon: 137.1687 }, izumo: { lat: 35.3667, lon: 132.7500 },
  kyoto: { lat: 35.0116, lon: 135.7681 }, yokohama: { lat: 35.4437, lon: 139.6380 },
  kobe: { lat: 34.6901, lon: 135.1955 }, sapporo: { lat: 43.0618, lon: 141.3545 },
  fukuoka: { lat: 33.5904, lon: 130.4017 }, hiroshima: { lat: 34.3853, lon: 132.4553 },
  sendai: { lat: 38.2682, lon: 140.8694 }, chiba: { lat: 35.6074, lon: 140.1065 },
  saitama: { lat: 35.8617, lon: 139.6455 }, shizuoka: { lat: 34.9756, lon: 138.3828 },
  toyota: { lat: 35.0833, lon: 137.1500 },
  // Brasil
  'sao paulo': { lat: -23.5505, lon: -46.6333 }, 'rio de janeiro': { lat: -22.9068, lon: -43.1729 },
  'belo horizonte': { lat: -19.9217, lon: -43.9378 }, salvador: { lat: -12.9714, lon: -38.5014 },
  curitiba: { lat: -25.4284, lon: -49.2733 }, 'porto alegre': { lat: -30.0346, lon: -51.2177 },
  recife: { lat: -8.0578, lon: -34.8829 }, fortaleza: { lat: -3.7319, lon: -38.5267 },
  brasilia: { lat: -15.7975, lon: -47.8919 }, goiania: { lat: -16.6869, lon: -49.2648 },
  florianopolis: { lat: -27.5954, lon: -48.5480 }, belem: { lat: -1.4558, lon: -48.4902 },
  manaus: { lat: -3.1190, lon: -60.0217 }, campinas: { lat: -22.9056, lon: -47.0608 },
  vitoria: { lat: -20.3155, lon: -40.3128 }, natal: { lat: -5.7945, lon: -35.2110 },
  'joao pessoa': { lat: -7.1195, lon: -34.8450 }, maceio: { lat: -9.6498, lon: -35.7089 },
  'campo grande': { lat: -20.4697, lon: -54.6201 }, cuiaba: { lat: -15.6010, lon: -56.0974 },
  // EUA
  'new york': { lat: 40.7128, lon: -74.0060 }, 'los angeles': { lat: 34.0522, lon: -118.2437 },
  chicago: { lat: 41.8781, lon: -87.6298 }, miami: { lat: 25.7617, lon: -80.1918 },
  'san francisco': { lat: 37.7749, lon: -122.4194 }, orlando: { lat: 28.5383, lon: -81.3792 },
  dallas: { lat: 32.7767, lon: -96.7970 }, houston: { lat: 29.7604, lon: -95.3698 },
  atlanta: { lat: 33.7490, lon: -84.3880 }, boston: { lat: 42.3601, lon: -71.0589 },
  seattle: { lat: 47.6062, lon: -122.3321 },
  // Europa / Outros
  lisbon: { lat: 38.7223, lon: -9.1393 }, lisboa: { lat: 38.7223, lon: -9.1393 },
  porto: { lat: 41.1579, lon: -8.6291 }, london: { lat: 51.5074, lon: -0.1278 },
  londres: { lat: 51.5074, lon: -0.1278 }, paris: { lat: 48.8566, lon: 2.3522 },
  madrid: { lat: 40.4168, lon: -3.7038 }, barcelona: { lat: 41.3851, lon: 2.1734 },
  berlin: { lat: 52.5200, lon: 13.4050 }, rome: { lat: 41.9028, lon: 12.4964 },
  roma: { lat: 41.9028, lon: 12.4964 }, amsterdam: { lat: 52.3676, lon: 4.9041 },
  dublin: { lat: 53.3498, lon: -6.2603 }, stockholm: { lat: 59.3293, lon: 18.0686 },
  singapore: { lat: 1.3521, lon: 103.8198 }, cingapura: { lat: 1.3521, lon: 103.8198 },
  'buenos aires': { lat: -34.6037, lon: -58.3816 }, 'mexico city': { lat: 19.4326, lon: -99.1332 },
  toronto: { lat: 43.6532, lon: -79.3832 }, sydney: { lat: -33.8688, lon: 151.2093 },
}

export const jpPrefectureCoords: Record<string, { lat: number; lon: number; name: string }> = {
  '13': { lat: 35.6895, lon: 139.6917, name: 'Tóquio' }, '27': { lat: 34.6937, lon: 135.5023, name: 'Osaka' },
  '23': { lat: 35.1815, lon: 136.9066, name: 'Aichi' }, '22': { lat: 34.9756, lon: 138.3828, name: 'Shizuoka' },
  '14': { lat: 35.4437, lon: 139.6380, name: 'Kanagawa' }, '28': { lat: 34.6901, lon: 135.1955, name: 'Hyogo' },
  '26': { lat: 35.0116, lon: 135.7681, name: 'Quioto' }, '01': { lat: 43.0618, lon: 141.3545, name: 'Hokkaido' },
  '40': { lat: 33.5904, lon: 130.4017, name: 'Fukuoka' }, '32': { lat: 35.4722, lon: 133.0505, name: 'Shimane' },
  '11': { lat: 35.8617, lon: 139.6455, name: 'Saitama' }, '12': { lat: 35.6074, lon: 140.1065, name: 'Chiba' },
  '34': { lat: 34.3853, lon: 132.4553, name: 'Hiroshima' }, '04': { lat: 38.2682, lon: 140.8694, name: 'Miyagi' },
  '47': { lat: 26.2124, lon: 127.6809, name: 'Okinawa' }, '08': { lat: 36.3659, lon: 140.4712, name: 'Ibaraki' },
  '09': { lat: 36.5658, lon: 139.8836, name: 'Tochigi' }, '10': { lat: 36.3907, lon: 139.0604, name: 'Gunma' },
  '15': { lat: 37.9022, lon: 139.0236, name: 'Niigata' }, '16': { lat: 36.6953, lon: 137.2113, name: 'Toyama' },
  '17': { lat: 36.5947, lon: 136.6256, name: 'Ishikawa' }, '18': { lat: 36.0652, lon: 136.2216, name: 'Fukui' },
  '19': { lat: 35.6642, lon: 138.5684, name: 'Yamanashi' }, '20': { lat: 36.6513, lon: 138.1810, name: 'Nagano' },
  '21': { lat: 35.4233, lon: 136.7607, name: 'Gifu' }, '24': { lat: 34.7303, lon: 136.5086, name: 'Mie' },
  '25': { lat: 35.0045, lon: 135.8686, name: 'Shiga' }, '29': { lat: 34.6853, lon: 135.8327, name: 'Nara' },
  '30': { lat: 34.2260, lon: 135.1675, name: 'Wakayama' }, '33': { lat: 34.6618, lon: 133.9344, name: 'Okayama' },
  '35': { lat: 34.1861, lon: 131.4705, name: 'Yamaguchi' }, '43': { lat: 32.7898, lon: 130.7417, name: 'Kumamoto' },
}

export const brStateCoords: Record<string, { lat: number; lon: number }> = {
  SP: { lat: -23.55, lon: -46.63 }, RJ: { lat: -22.91, lon: -43.17 }, MG: { lat: -19.92, lon: -43.93 },
  BA: { lat: -12.98, lon: -38.50 }, RS: { lat: -30.03, lon: -51.22 }, PR: { lat: -25.43, lon: -49.27 },
  PE: { lat: -8.06, lon: -34.88 }, CE: { lat: -3.73, lon: -38.53 }, PA: { lat: -1.46, lon: -48.49 },
  SC: { lat: -27.60, lon: -48.55 }, MA: { lat: -2.53, lon: -44.31 }, GO: { lat: -16.69, lon: -49.26 },
  AM: { lat: -3.12, lon: -60.02 }, DF: { lat: -15.79, lon: -47.88 }, ES: { lat: -20.32, lon: -40.34 },
  MT: { lat: -15.60, lon: -56.10 }, MS: { lat: -20.47, lon: -54.62 }, RN: { lat: -5.79, lon: -35.21 },
  PB: { lat: -7.12, lon: -34.85 }, AL: { lat: -9.65, lon: -35.71 }, SE: { lat: -10.95, lon: -37.07 },
  PI: { lat: -5.09, lon: -42.80 }, TO: { lat: -10.18, lon: -48.33 }, RO: { lat: -8.76, lon: -63.90 },
  AC: { lat: -9.97, lon: -67.81 }, AP: { lat: 0.03, lon: -51.05 }, RR: { lat: 2.82, lon: -60.67 },
}

export const usStateCoords: Record<string, { lat: number; lon: number }> = {
  CA: { lat: 36.7783, lon: -119.4179 }, NY: { lat: 40.7128, lon: -74.0060 }, FL: { lat: 27.6648, lon: -81.5158 },
  TX: { lat: 31.9686, lon: -99.9018 }, IL: { lat: 40.6331, lon: -89.3985 }, WA: { lat: 47.7511, lon: -120.7401 },
  MA: { lat: 42.4072, lon: -71.3824 }, PA: { lat: 41.2033, lon: -77.1945 }, GA: { lat: 32.1656, lon: -82.9001 },
  NC: { lat: 35.7596, lon: -79.0193 }, NJ: { lat: 40.0583, lon: -74.4057 }, OH: { lat: 40.4173, lon: -82.9071 },
  VA: { lat: 37.4316, lon: -78.6569 }, CO: { lat: 39.5501, lon: -105.7821 }, AZ: { lat: 34.0489, lon: -111.0937 },
  NV: { lat: 38.8026, lon: -116.4194 },
}

export const countryInfo: Record<string, { lat: number; lon: number; name: string; flag: string }> = {
  BR: { lat: -14.235, lon: -51.925, name: 'Brasil', flag: '🇧🇷' },
  JP: { lat: 36.2048, lon: 138.2529, name: 'Japão', flag: '🇯🇵' },
  US: { lat: 37.0902, lon: -95.7129, name: 'Estados Unidos', flag: '🇺🇸' },
  SG: { lat: 1.3521, lon: 103.8198, name: 'Cingapura', flag: '🇸🇬' },
  SE: { lat: 60.1282, lon: 18.6435, name: 'Suécia', flag: '🇸🇪' },
  IE: { lat: 53.1424, lon: -7.6921, name: 'Irlanda', flag: '🇮🇪' },
  CN: { lat: 35.8617, lon: 104.1954, name: 'China', flag: '🇨🇳' },
  FR: { lat: 46.2276, lon: 2.2137, name: 'França', flag: '🇫🇷' },
  NL: { lat: 52.1326, lon: 5.2913, name: 'Holanda', flag: '🇳🇱' },
  AR: { lat: -38.4161, lon: -63.6167, name: 'Argentina', flag: '🇦🇷' },
  PL: { lat: 51.9194, lon: 19.1451, name: 'Polônia', flag: '🇵🇱' },
  CA: { lat: 56.1304, lon: -106.3468, name: 'Canadá', flag: '🇨🇦' },
  DE: { lat: 51.1657, lon: 10.4515, name: 'Alemanha', flag: '🇩🇪' },
  FI: { lat: 61.9241, lon: 25.7482, name: 'Finlândia', flag: '🇫🇮' },
  GB: { lat: 55.3781, lon: -3.4360, name: 'Reino Unido', flag: '🇬🇧' },
  PT: { lat: 39.3999, lon: -8.2245, name: 'Portugal', flag: '🇵🇹' },
  ES: { lat: 40.4637, lon: -3.7492, name: 'Espanha', flag: '🇪🇸' },
  IT: { lat: 41.8719, lon: 12.5674, name: 'Itália', flag: '🇮🇹' },
  MX: { lat: 23.6345, lon: -102.5528, name: 'México', flag: '🇲🇽' },
  AU: { lat: -25.2744, lon: 133.7751, name: 'Austrália', flag: '🇦🇺' },
  CL: { lat: -35.6751, lon: -71.5430, name: 'Chile', flag: '🇨🇱' },
  CO: { lat: 4.5709, lon: -74.2973, name: 'Colômbia', flag: '🇨🇴' },
  PE: { lat: -9.1900, lon: -75.0152, name: 'Peru', flag: '🇵🇪' },
  UY: { lat: -32.5228, lon: -55.7658, name: 'Uruguai', flag: '🇺🇾' },
  PY: { lat: -23.4425, lon: -58.4438, name: 'Paraguai', flag: '🇵🇾' },
  IN: { lat: 20.5937, lon: 78.9629, name: 'Índia', flag: '🇮🇳' },
  KR: { lat: 35.9078, lon: 127.7669, name: 'Coreia do Sul', flag: '🇰🇷' },
  ZA: { lat: -30.5595, lon: 22.9375, name: 'África do Sul', flag: '🇿🇦' },
}

export const regionFocusList: RegionFocus[] = [
  { id: 'all', label: '🌎 Global', targetLon: -50, targetLat: -10 },
  { id: 'br', label: '🇧🇷 Brasil', targetLon: -51.9, targetLat: -14.2 },
  { id: 'jp', label: '🇯🇵 Japão', targetLon: 138.2, targetLat: 36.2 },
  { id: 'us', label: '🇺🇸 EUA', targetLon: -95.7, targetLat: 37.0 },
  { id: 'eu', label: '🇪🇺 Europa', targetLon: 10.0, targetLat: 50.0 },
]

export function applyJitter(coord: { lat: number; lon: number }, seed: string) {
  let hash = 0
  for (let i = 0; i < seed.length; i++) hash = seed.charCodeAt(i) + ((hash << 5) - hash)
  const offsetLat = ((hash % 100) / 100) * 1.2 - 0.6
  const offsetLon = (((hash >> 8) % 100) / 100) * 1.2 - 0.6
  return {
    lat: Math.max(-85, Math.min(85, coord.lat + offsetLat)),
    lon: Math.max(-180, Math.min(180, coord.lon + offsetLon)),
  }
}

export function capitalize(str: string) {
  return str ? str.charAt(0).toUpperCase() + str.slice(1) : ''
}

export function formatCompactNumber(num: number): string {
  if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + 'M'
  if (num >= 1_000) return (num / 1_000).toFixed(1) + 'k'
  return num.toString()
}

export function resolveCoordinates(s: GeoStat): { lat: number; lon: number; resolvedName: string; flag: string } {
  const country = (s.country || '').trim().toUpperCase()
  const state = (s.state || '').trim().toUpperCase()
  const city = (s.city || '').trim().toLowerCase()
  const countryMeta = countryInfo[country] || { lat: -14.235, lon: -51.925, name: country || 'Mundo', flag: '🌐' }

  if (city && cityCoords[city]) {
    const coords = applyJitter(cityCoords[city], city + state)
    return { ...coords, resolvedName: s.city ? `${capitalize(s.city)}, ${countryMeta.name}` : countryMeta.name, flag: countryMeta.flag }
  }
  if (country === 'JP') {
    if (state && jpPrefectureCoords[state]) {
      const p = jpPrefectureCoords[state]!
      return { ...applyJitter(p, state + city), resolvedName: `${p.name}, Japão`, flag: '🇯🇵' }
    }
    const jpMeta = countryInfo.JP!
    return { ...applyJitter(jpMeta, city || state || 'JP'), resolvedName: city ? `${capitalize(city)}, Japão` : 'Japão', flag: '🇯🇵' }
  }
  if (country === 'BR' || (!country && brStateCoords[state])) {
    if (state && brStateCoords[state]) {
      return { ...applyJitter(brStateCoords[state]!, state + city), resolvedName: city ? `${capitalize(city)} (${state}), Brasil` : `Estado de ${state}, Brasil`, flag: '🇧🇷' }
    }
    const brMeta = countryInfo.BR!
    return { ...applyJitter(brMeta, city || state || 'BR'), resolvedName: city ? `${capitalize(city)}, Brasil` : 'Brasil', flag: '🇧🇷' }
  }
  if (country === 'US') {
    if (state && usStateCoords[state]) {
      return { ...applyJitter(usStateCoords[state]!, state + city), resolvedName: city ? `${capitalize(city)} (${state}), EUA` : `${state}, EUA`, flag: '🇺🇸' }
    }
    const usMeta = countryInfo.US!
    return { ...applyJitter(usMeta, city || state || 'US'), resolvedName: city ? `${capitalize(city)}, EUA` : 'Estados Unidos', flag: '🇺🇸' }
  }
  const baseState = brStateCoords[state] || usStateCoords[state]
  if (state && baseState) {
    return { ...applyJitter(baseState, state), resolvedName: state, flag: countryMeta.flag }
  }
  if (country && countryInfo[country]) {
    return { ...applyJitter(countryInfo[country]!, city || state || country), resolvedName: city ? `${capitalize(city)}, ${countryMeta.name}` : countryMeta.name, flag: countryMeta.flag }
  }
  return { ...applyJitter(countryInfo.BR!, 'default'), resolvedName: 'Localização Desconhecida', flag: '🌐' }
}
