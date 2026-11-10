package traffic

import "strings"

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func isAdVerifierUA(ua string) bool {
	return containsAny(ua,
		"meta-external", "facebookexternalhit", "facebookcatalog", "facebookplatform",
		"facebot", "adsbot-google", "google-adwords-instant", "mediapartners-google",
		"feedfetcher-google", "tiktokbot", "twitterbot", "telegrambot", "whatsapp",
		"linkedinbot", "pinterestbot", "slackbot", "discordbot",
	)
}

func isSearchBotUA(ua string) bool {
	return containsAny(ua,
		"googlebot", "bingbot", "yandexbot", "duckduckbot", "baiduspider", "sogou", "exabot",
	)
}

func isHeadlessScraperUA(ua string) bool {
	return containsAny(ua,
		"headlesschrome", "phantomjs", "playwright", "puppeteer", "selenium", "electron",
		"curl/", "wget/", "postman", "python-requests", "python", "aiohttp", "scrapy",
		"bytespider", "bytedance", "gptbot", "chatgpt-user", "claudebot", "claude-web",
		"anthropic-ai", "perplexitybot", "semrushbot", "ahrefsbot", "applebot", "petalbot", "dotbot",
	)
}

func isGenericBotUA(ua string) bool {
	return containsAny(ua, "bot", "spider", "crawler", "crawl", "slurp", "scraper", "fetcher")
}

func isSocialOrSearchHost(host string) bool {
	return containsAny(host,
		"instagram.com", "facebook.com", "fb.com", "google.", "tiktok.com",
		"youtube.com", "whatsapp.com", "linkedin.com", "t.co", "bing.com",
	)
}

func detectBrowser(ua string) string {
	switch {
	case containsAny(ua, "meta-external", "facebookexternalhit", "facebot"):
		return "Meta / Facebook Crawler"
	case containsAny(ua, "googlebot", "adsbot-google", "mediapartners-google"):
		return "Google Crawler"
	case containsAny(ua, "tiktokbot", "bytespider"):
		return "TikTok / ByteDance Bot"
	case strings.Contains(ua, "bingbot"):
		return "Bing Crawler"
	case containsAny(ua, "gptbot", "chatgpt"):
		return "ChatGPT Bot"
	case containsAny(ua, "claudebot", "claude-web", "anthropic"):
		return "Anthropic Claude Bot"
	case isGenericBotUA(ua):
		return "Robô / Rastreador Web"
	case strings.Contains(ua, "edg/"):
		return "Microsoft Edge"
	case containsAny(ua, "opr/", "opera/"):
		return "Opera"
	case strings.Contains(ua, "samsungbrowser"):
		return "Samsung Internet"
	case strings.Contains(ua, "instagram"):
		return "Instagram In-App Browser"
	case containsAny(ua, "fbav", "fban"):
		return "Facebook In-App Browser"
	case strings.Contains(ua, "chrome/"):
		return "Google Chrome"
	case strings.Contains(ua, "safari/") && !strings.Contains(ua, "chrome/"):
		return "Apple Safari"
	case strings.Contains(ua, "firefox/"):
		return "Mozilla Firefox"
	default:
		return "Navegador Desconhecido"
	}
}

func detectOS(ua string) string {
	switch {
	case containsAny(ua, "meta-external", "facebookexternalhit", "googlebot") || isGenericBotUA(ua):
		return "Servidor Automatizado"
	case strings.Contains(ua, "windows nt"):
		return "Windows"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone"):
		return "iOS (iPhone)"
	case strings.Contains(ua, "ipad"):
		return "iPadOS"
	case containsAny(ua, "macintosh", "mac os x"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "SO Desconhecido"
	}
}

func detectDevice(ua string) string {
	switch {
	case containsAny(ua, "meta-external", "facebookexternalhit", "googlebot") || isGenericBotUA(ua):
		return "Servidor / Robô"
	case containsAny(ua, "ipad", "tablet"):
		return "Tablet"
	case containsAny(ua, "mobile", "android", "iphone"):
		return "Celular"
	default:
		return "Computador"
	}
}

