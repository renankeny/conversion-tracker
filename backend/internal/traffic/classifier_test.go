package traffic

import (
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name         string
		ua           string
		referrer     string
		isWebdriver  bool
		wantType     string
		wantCategory string
		wantMinScore int
	}{
		{
			name:         "Direct access with Chrome on Windows",
			ua:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			referrer:     "",
			isWebdriver:  false,
			wantType:     "human",
			wantCategory: "none",
			wantMinScore: 60,
		},
		{
			name:         "Instagram Ad click on iPhone",
			ua:           "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 345.0.0",
			referrer:     "https://l.instagram.com/",
			isWebdriver:  false,
			wantType:     "human",
			wantCategory: "none",
			wantMinScore: 65,
		},
		{
			name:         "Meta Ads Crawler (facebookexternalhit)",
			ua:           "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
			referrer:     "",
			isWebdriver:  false,
			wantType:     "bot",
			wantCategory: "ad_verifier",
			wantMinScore: -80,
		},
		{
			name:         "Google Ads Bot",
			ua:           "AdsBot-Google (+http://www.google.com/adsbot.html)",
			referrer:     "",
			isWebdriver:  false,
			wantType:     "bot",
			wantCategory: "ad_verifier",
			wantMinScore: -80,
		},
		{
			name:         "Headless Chrome with webdriver active",
			ua:           "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36",
			referrer:     "",
			isWebdriver:  true,
			wantType:     "bot",
			wantCategory: "headless_scraper",
			wantMinScore: -90,
		},
		{
			name:         "Empty / Truncated UA",
			ua:           "curl/7.68.0",
			referrer:     "",
			isWebdriver:  false,
			wantType:     "bot",
			wantCategory: "headless_scraper",
			wantMinScore: -90,
		},
		{
			name:         "Inconclusive generic UA",
			ua:           "Mozilla/5.0 (Unknown; U; CustomEngine)",
			referrer:     "",
			isWebdriver:  false,
			wantType:     "inconclusive",
			wantCategory: "none",
			wantMinScore: -30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Classify(tt.ua, tt.referrer, tt.isWebdriver)
			if res.Type != tt.wantType {
				t.Errorf("Classify().Type = %v, want %v", res.Type, tt.wantType)
			}
			if res.Category != tt.wantCategory {
				t.Errorf("Classify().Category = %v, want %v", res.Category, tt.wantCategory)
			}
			if res.Disclaimer == "" {
				t.Errorf("Classify().Disclaimer should not be empty")
			}
			if len(res.Rules) == 0 {
				t.Errorf("Classify().Rules should have at least one triggered rule")
			}
		})
	}
}
