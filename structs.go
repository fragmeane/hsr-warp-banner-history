package main

type Response struct {
	Config struct {
		Banners map[int]struct {
			RateUp          int    `json:"rateup"`
			Sort            int    `json:"sort"`
			Rerun           int    `json:"rerun"`
			Accent          string `json:"accent"`
			Border          string `json:"border"`
			RateUp4         []int  `json:"rateup4"`
			IsExclusive     bool   `json:"is_exclusive"`
			StartTime       int    `json:"start_time"`
			EndTime         int    `json:"end_time"`
			CompanionBanner int    `json:"companion_banner"`
		}
		Cost           int
		Features       []string
		IsMaintenance  bool   `json:"is_maintenance"`
		MaintenanceMsg string `json:"maintenance_msg"`
		Sort           []int
		Types          map[any]any `json:"-"`
		WebCacheVer    string      `json:"webcache_ver"`
	}
	Time      int
	CompatVer int `json:"compat_ver"`
}

type BannerHistory struct {
	Banners []Banner `json:"banners"`
}

type Banner struct {
	Desc            string `json:"desc"`
	GachaType       string `json:"gacha_type"`
	Id              string `json:"id"`
	Version         string `json:"version"`
	RateUp4         []int  `json:"rateup4"`
	CompanionBanner int    `json:"companion_banner"`
	EndTime         int    `json:"end_time"`
	RateUp          int    `json:"rateup"`
	Rerun           int    `json:"rerun"`
	StartTime       int    `json:"start_time"`
	IsExclusive     bool   `json:"is_exclusive"`
}

type LightCone struct {
	Desc     string
	Icon     string
	Id       string
	Name     string
	Path     string
	Portrait string
	Preview  string
	Rarity   int
}
