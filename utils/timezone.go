package utils

// TimezoneInfo содержит информацию о часовом поясе
type TimezoneInfo struct {
	ID          string
	Name        string
	Offset      string
	Region      string
	CountryCode string // ISO 3166-1 alpha-2 country code
}

// GetAvailableTimezones возвращает список доступных часовых поясов
func GetAvailableTimezones() []TimezoneInfo {
	return []TimezoneInfo{
		// Стандартные
		{ID: "UTC", Name: "UTC", Offset: "+00:00", Region: "Universal", CountryCode: "UN"},

		// Европа
		{ID: "Europe/Moscow", Name: "Moscow", Offset: "+03:00", Region: "Europe", CountryCode: "RU"},
		{ID: "Europe/London", Name: "London", Offset: "+00:00", Region: "Europe", CountryCode: "GB"},
		{ID: "Europe/Paris", Name: "Paris", Offset: "+01:00", Region: "Europe", CountryCode: "FR"},
		{ID: "Europe/Berlin", Name: "Berlin", Offset: "+01:00", Region: "Europe", CountryCode: "DE"},
		{ID: "Europe/Kiev", Name: "Kiev", Offset: "+02:00", Region: "Europe", CountryCode: "UA"},
		{ID: "Europe/Madrid", Name: "Madrid", Offset: "+01:00", Region: "Europe", CountryCode: "ES"},
		{ID: "Europe/Rome", Name: "Rome", Offset: "+01:00", Region: "Europe", CountryCode: "IT"},
		{ID: "Europe/Athens", Name: "Athens", Offset: "+02:00", Region: "Europe", CountryCode: "GR"},
		{ID: "Europe/Istanbul", Name: "Istanbul", Offset: "+03:00", Region: "Europe", CountryCode: "TR"},
		{ID: "Europe/Warsaw", Name: "Warsaw", Offset: "+01:00", Region: "Europe", CountryCode: "PL"},
		{ID: "Europe/Amsterdam", Name: "Amsterdam", Offset: "+01:00", Region: "Europe", CountryCode: "NL"},
		{ID: "Europe/Stockholm", Name: "Stockholm", Offset: "+01:00", Region: "Europe", CountryCode: "SE"},
		{ID: "Europe/Vienna", Name: "Vienna", Offset: "+01:00", Region: "Europe", CountryCode: "AT"},
		{ID: "Europe/Minsk", Name: "Minsk", Offset: "+03:00", Region: "Europe", CountryCode: "BY"},
		{ID: "Europe/Dublin", Name: "Dublin", Offset: "+00:00", Region: "Europe", CountryCode: "IE"},
		{ID: "Europe/Brussels", Name: "Brussels", Offset: "+01:00", Region: "Europe", CountryCode: "BE"},
		{ID: "Europe/Lisbon", Name: "Lisbon", Offset: "+00:00", Region: "Europe", CountryCode: "PT"},
		{ID: "Europe/Bucharest", Name: "Bucharest", Offset: "+02:00", Region: "Europe", CountryCode: "RO"},
		{ID: "Europe/Budapest", Name: "Budapest", Offset: "+01:00", Region: "Europe", CountryCode: "HU"},
		{ID: "Europe/Prague", Name: "Prague", Offset: "+01:00", Region: "Europe", CountryCode: "CZ"},
		{ID: "Europe/Sofia", Name: "Sofia", Offset: "+02:00", Region: "Europe", CountryCode: "BG"},
		{ID: "Europe/Copenhagen", Name: "Copenhagen", Offset: "+01:00", Region: "Europe", CountryCode: "DK"},
		{ID: "Europe/Helsinki", Name: "Helsinki", Offset: "+02:00", Region: "Europe", CountryCode: "FI"},
		{ID: "Europe/Oslo", Name: "Oslo", Offset: "+01:00", Region: "Europe", CountryCode: "NO"},
		{ID: "Europe/Riga", Name: "Riga", Offset: "+02:00", Region: "Europe", CountryCode: "LV"},
		{ID: "Europe/Tallinn", Name: "Tallinn", Offset: "+02:00", Region: "Europe", CountryCode: "EE"},
		{ID: "Europe/Vilnius", Name: "Vilnius", Offset: "+02:00", Region: "Europe", CountryCode: "LT"},
		{ID: "Europe/Belgrade", Name: "Belgrade", Offset: "+01:00", Region: "Europe", CountryCode: "RS"},
		{ID: "Europe/Ljubljana", Name: "Ljubljana", Offset: "+01:00", Region: "Europe", CountryCode: "SI"},
		{ID: "Europe/Bratislava", Name: "Bratislava", Offset: "+01:00", Region: "Europe", CountryCode: "SK"},
		{ID: "Europe/Zagreb", Name: "Zagreb", Offset: "+01:00", Region: "Europe", CountryCode: "HR"},
		{ID: "Europe/Skopje", Name: "Skopje", Offset: "+01:00", Region: "Europe", CountryCode: "MK"},
		{ID: "Europe/Sarajevo", Name: "Sarajevo", Offset: "+01:00", Region: "Europe", CountryCode: "BA"},
		{ID: "Europe/Podgorica", Name: "Podgorica", Offset: "+01:00", Region: "Europe", CountryCode: "ME"},
		{ID: "Europe/Chisinau", Name: "Chisinau", Offset: "+02:00", Region: "Europe", CountryCode: "MD"},
		{ID: "Europe/Monaco", Name: "Monaco", Offset: "+01:00", Region: "Europe", CountryCode: "MC"},
		{ID: "Europe/Vaduz", Name: "Vaduz", Offset: "+01:00", Region: "Europe", CountryCode: "LI"},
		{ID: "Europe/Luxembourg", Name: "Luxembourg", Offset: "+01:00", Region: "Europe", CountryCode: "LU"},
		{ID: "Europe/Andorra", Name: "Andorra", Offset: "+01:00", Region: "Europe", CountryCode: "AD"},
		{ID: "Europe/Malta", Name: "Malta", Offset: "+01:00", Region: "Europe", CountryCode: "MT"},
		{ID: "Europe/San_Marino", Name: "San Marino", Offset: "+01:00", Region: "Europe", CountryCode: "SM"},
		{ID: "Europe/Vatican", Name: "Vatican", Offset: "+01:00", Region: "Europe", CountryCode: "VA"},

		// Америка
		{ID: "America/New_York", Name: "New York", Offset: "-05:00", Region: "America", CountryCode: "US"},
		{ID: "America/Los_Angeles", Name: "Los Angeles", Offset: "-08:00", Region: "America", CountryCode: "US"},
		{ID: "America/Chicago", Name: "Chicago", Offset: "-06:00", Region: "America", CountryCode: "US"},
		{ID: "America/Denver", Name: "Denver", Offset: "-07:00", Region: "America", CountryCode: "US"},
		{ID: "America/Phoenix", Name: "Phoenix", Offset: "-07:00", Region: "America", CountryCode: "US"},
		{ID: "America/Toronto", Name: "Toronto", Offset: "-05:00", Region: "America", CountryCode: "CA"},
		{ID: "America/Vancouver", Name: "Vancouver", Offset: "-08:00", Region: "America", CountryCode: "CA"},
		{ID: "America/Mexico_City", Name: "Mexico City", Offset: "-06:00", Region: "America", CountryCode: "MX"},
		{ID: "America/Sao_Paulo", Name: "Sao Paulo", Offset: "-03:00", Region: "America", CountryCode: "BR"},
		{ID: "America/Buenos_Aires", Name: "Buenos Aires", Offset: "-03:00", Region: "America", CountryCode: "AR"},
		{ID: "America/Santiago", Name: "Santiago", Offset: "-04:00", Region: "America", CountryCode: "CL"},
		{ID: "America/Bogota", Name: "Bogota", Offset: "-05:00", Region: "America", CountryCode: "CO"},
		{ID: "America/Lima", Name: "Lima", Offset: "-05:00", Region: "America", CountryCode: "PE"},
		{ID: "America/Caracas", Name: "Caracas", Offset: "-04:00", Region: "America", CountryCode: "VE"},
		{ID: "America/Halifax", Name: "Halifax", Offset: "-04:00", Region: "America", CountryCode: "CA"},
		{ID: "America/Washington", Name: "Washington", Offset: "-05:00", Region: "America", CountryCode: "US"},
		{ID: "America/Ottawa", Name: "Ottawa", Offset: "-05:00", Region: "America", CountryCode: "CA"},
		{ID: "America/Havana", Name: "Havana", Offset: "-05:00", Region: "America", CountryCode: "CU"},
		{ID: "America/Port_au_Prince", Name: "Port-au-Prince", Offset: "-05:00", Region: "America", CountryCode: "HT"},
		{ID: "America/Santo_Domingo", Name: "Santo Domingo", Offset: "-04:00", Region: "America", CountryCode: "DO"},
		{ID: "America/Guatemala", Name: "Guatemala", Offset: "-06:00", Region: "America", CountryCode: "GT"},
		{ID: "America/Tegucigalpa", Name: "Tegucigalpa", Offset: "-06:00", Region: "America", CountryCode: "HN"},
		{ID: "America/Managua", Name: "Managua", Offset: "-06:00", Region: "America", CountryCode: "NI"},
		{ID: "America/San_Salvador", Name: "San Salvador", Offset: "-06:00", Region: "America", CountryCode: "SV"},
		{ID: "America/Panama", Name: "Panama", Offset: "-05:00", Region: "America", CountryCode: "PA"},
		{ID: "America/Belmopan", Name: "Belmopan", Offset: "-06:00", Region: "America", CountryCode: "BZ"},
		{ID: "America/San_Jose", Name: "San Jose", Offset: "-06:00", Region: "America", CountryCode: "CR"},
		{ID: "America/Kingston", Name: "Kingston", Offset: "-05:00", Region: "America", CountryCode: "JM"},
		{ID: "America/Nassau", Name: "Nassau", Offset: "-05:00", Region: "America", CountryCode: "BS"},
		{ID: "America/La_Paz", Name: "La Paz", Offset: "-04:00", Region: "America", CountryCode: "BO"},
		{ID: "America/Asuncion", Name: "Asuncion", Offset: "-04:00", Region: "America", CountryCode: "PY"},
		{ID: "America/Montevideo", Name: "Montevideo", Offset: "-03:00", Region: "America", CountryCode: "UY"},
		{ID: "America/Paramaribo", Name: "Paramaribo", Offset: "-03:00", Region: "America", CountryCode: "SR"},
		{ID: "America/Georgetown", Name: "Georgetown", Offset: "-04:00", Region: "America", CountryCode: "GY"},
		{ID: "America/Quito", Name: "Quito", Offset: "-05:00", Region: "America", CountryCode: "EC"},
		{ID: "America/Bridgetown", Name: "Bridgetown", Offset: "-04:00", Region: "America", CountryCode: "BB"},
		{ID: "America/Port_of_Spain", Name: "Port of Spain", Offset: "-04:00", Region: "America", CountryCode: "TT"},
		{ID: "America/St_Johns", Name: "St. John's", Offset: "-03:30", Region: "America", CountryCode: "CA"},
		{ID: "America/Brasilia", Name: "Brasilia", Offset: "-03:00", Region: "America", CountryCode: "BR"},

		// Азия
		{ID: "Asia/Tokyo", Name: "Tokyo", Offset: "+09:00", Region: "Asia", CountryCode: "JP"},
		{ID: "Asia/Shanghai", Name: "Shanghai", Offset: "+08:00", Region: "Asia", CountryCode: "CN"},
		{ID: "Asia/Hong_Kong", Name: "Hong Kong", Offset: "+08:00", Region: "Asia", CountryCode: "HK"},
		{ID: "Asia/Singapore", Name: "Singapore", Offset: "+08:00", Region: "Asia", CountryCode: "SG"},
		{ID: "Asia/Seoul", Name: "Seoul", Offset: "+09:00", Region: "Asia", CountryCode: "KR"},
		{ID: "Asia/Dubai", Name: "Dubai", Offset: "+04:00", Region: "Asia", CountryCode: "AE"},
		{ID: "Asia/Bangkok", Name: "Bangkok", Offset: "+07:00", Region: "Asia", CountryCode: "TH"},
		{ID: "Asia/Kolkata", Name: "New Delhi", Offset: "+05:30", Region: "Asia", CountryCode: "IN"},
		{ID: "Asia/Jakarta", Name: "Jakarta", Offset: "+07:00", Region: "Asia", CountryCode: "ID"},
		{ID: "Asia/Manila", Name: "Manila", Offset: "+08:00", Region: "Asia", CountryCode: "PH"},
		{ID: "Asia/Taipei", Name: "Taipei", Offset: "+08:00", Region: "Asia", CountryCode: "TW"},
		{ID: "Asia/Riyadh", Name: "Riyadh", Offset: "+03:00", Region: "Asia", CountryCode: "SA"},
		{ID: "Asia/Tel_Aviv", Name: "Tel Aviv", Offset: "+02:00", Region: "Asia", CountryCode: "IL"},
		{ID: "Asia/Tehran", Name: "Tehran", Offset: "+03:30", Region: "Asia", CountryCode: "IR"},
		{ID: "Asia/Baghdad", Name: "Baghdad", Offset: "+03:00", Region: "Asia", CountryCode: "IQ"},
		{ID: "Asia/Beijing", Name: "Beijing", Offset: "+08:00", Region: "Asia", CountryCode: "CN"},
		{ID: "Asia/Islamabad", Name: "Islamabad", Offset: "+05:00", Region: "Asia", CountryCode: "PK"},
		{ID: "Asia/Kabul", Name: "Kabul", Offset: "+04:30", Region: "Asia", CountryCode: "AF"},
		{ID: "Asia/Tashkent", Name: "Tashkent", Offset: "+05:00", Region: "Asia", CountryCode: "UZ"},
		{ID: "Asia/Ashgabat", Name: "Ashgabat", Offset: "+05:00", Region: "Asia", CountryCode: "TM"},
		{ID: "Asia/Dushanbe", Name: "Dushanbe", Offset: "+05:00", Region: "Asia", CountryCode: "TJ"},
		{ID: "Asia/Bishkek", Name: "Bishkek", Offset: "+06:00", Region: "Asia", CountryCode: "KG"},
		{ID: "Asia/Astana", Name: "Astana", Offset: "+06:00", Region: "Asia", CountryCode: "KZ"},
		{ID: "Asia/Kuala_Lumpur", Name: "Kuala Lumpur", Offset: "+08:00", Region: "Asia", CountryCode: "MY"},
		{ID: "Asia/Hanoi", Name: "Hanoi", Offset: "+07:00", Region: "Asia", CountryCode: "VN"},
		{ID: "Asia/Phnom_Penh", Name: "Phnom Penh", Offset: "+07:00", Region: "Asia", CountryCode: "KH"},
		{ID: "Asia/Vientiane", Name: "Vientiane", Offset: "+07:00", Region: "Asia", CountryCode: "LA"},
		{ID: "Asia/Yangon", Name: "Yangon", Offset: "+06:30", Region: "Asia", CountryCode: "MM"},
		{ID: "Asia/Dhaka", Name: "Dhaka", Offset: "+06:00", Region: "Asia", CountryCode: "BD"},
		{ID: "Asia/Thimphu", Name: "Thimphu", Offset: "+06:00", Region: "Asia", CountryCode: "BT"},
		{ID: "Asia/Kathmandu", Name: "Kathmandu", Offset: "+05:45", Region: "Asia", CountryCode: "NP"},
		{ID: "Asia/Colombo", Name: "Colombo", Offset: "+05:30", Region: "Asia", CountryCode: "LK"},
		{ID: "Asia/Ulaanbaatar", Name: "Ulaanbaatar", Offset: "+08:00", Region: "Asia", CountryCode: "MN"},
		{ID: "Asia/Pyongyang", Name: "Pyongyang", Offset: "+09:00", Region: "Asia", CountryCode: "KP"},
		{ID: "Asia/Muscat", Name: "Muscat", Offset: "+04:00", Region: "Asia", CountryCode: "OM"},
		{ID: "Asia/Qatar", Name: "Doha", Offset: "+03:00", Region: "Asia", CountryCode: "QA"},
		{ID: "Asia/Kuwait", Name: "Kuwait City", Offset: "+03:00", Region: "Asia", CountryCode: "KW"},
		{ID: "Asia/Bahrain", Name: "Manama", Offset: "+03:00", Region: "Asia", CountryCode: "BH"},
		{ID: "Asia/Amman", Name: "Amman", Offset: "+02:00", Region: "Asia", CountryCode: "JO"},
		{ID: "Asia/Beirut", Name: "Beirut", Offset: "+02:00", Region: "Asia", CountryCode: "LB"},
		{ID: "Asia/Damascus", Name: "Damascus", Offset: "+02:00", Region: "Asia", CountryCode: "SY"},
		{ID: "Asia/Jerusalem", Name: "Jerusalem", Offset: "+02:00", Region: "Asia", CountryCode: "IL"},
		{ID: "Asia/Baku", Name: "Baku", Offset: "+04:00", Region: "Asia", CountryCode: "AZ"},
		{ID: "Asia/Yerevan", Name: "Yerevan", Offset: "+04:00", Region: "Asia", CountryCode: "AM"},
		{ID: "Asia/Tbilisi", Name: "Tbilisi", Offset: "+04:00", Region: "Asia", CountryCode: "GE"},

		// Океания и Австралия
		{ID: "Australia/Sydney", Name: "Sydney", Offset: "+10:00", Region: "Australia", CountryCode: "AU"},
		{ID: "Australia/Melbourne", Name: "Melbourne", Offset: "+10:00", Region: "Australia", CountryCode: "AU"},
		{ID: "Australia/Brisbane", Name: "Brisbane", Offset: "+10:00", Region: "Australia", CountryCode: "AU"},
		{ID: "Australia/Perth", Name: "Perth", Offset: "+08:00", Region: "Australia", CountryCode: "AU"},
		{ID: "Australia/Adelaide", Name: "Adelaide", Offset: "+09:30", Region: "Australia", CountryCode: "AU"},
		{ID: "Australia/Canberra", Name: "Canberra", Offset: "+10:00", Region: "Australia", CountryCode: "AU"},
		{ID: "Pacific/Auckland", Name: "Auckland", Offset: "+12:00", Region: "Pacific", CountryCode: "NZ"},
		{ID: "Pacific/Fiji", Name: "Suva", Offset: "+12:00", Region: "Pacific", CountryCode: "FJ"},
		{ID: "Pacific/Honolulu", Name: "Honolulu", Offset: "-10:00", Region: "Pacific", CountryCode: "US"},
		{ID: "Pacific/Guam", Name: "Guam", Offset: "+10:00", Region: "Pacific", CountryCode: "GU"},
		{ID: "Pacific/Port_Moresby", Name: "Port Moresby", Offset: "+10:00", Region: "Pacific", CountryCode: "PG"},
		{ID: "Pacific/Apia", Name: "Apia", Offset: "+13:00", Region: "Pacific", CountryCode: "WS"},
		{ID: "Pacific/Tarawa", Name: "Tarawa", Offset: "+12:00", Region: "Pacific", CountryCode: "KI"},
		{ID: "Pacific/Funafuti", Name: "Funafuti", Offset: "+12:00", Region: "Pacific", CountryCode: "TV"},
		{ID: "Pacific/Majuro", Name: "Majuro", Offset: "+12:00", Region: "Pacific", CountryCode: "MH"},
		{ID: "Pacific/Yaren", Name: "Yaren", Offset: "+12:00", Region: "Pacific", CountryCode: "NR"},
		{ID: "Pacific/Palau", Name: "Ngerulmud", Offset: "+09:00", Region: "Pacific", CountryCode: "PW"},
		{ID: "Pacific/Honiara", Name: "Honiara", Offset: "+11:00", Region: "Pacific", CountryCode: "SB"},
		{ID: "Pacific/Noumea", Name: "Noumea", Offset: "+11:00", Region: "Pacific", CountryCode: "NC"},
		{ID: "Pacific/Pago_Pago", Name: "Pago Pago", Offset: "-11:00", Region: "Pacific", CountryCode: "AS"},
		{ID: "Pacific/Nuku_alofa", Name: "Nuku'alofa", Offset: "+13:00", Region: "Pacific", CountryCode: "TO"},
		{ID: "Pacific/Pohnpei", Name: "Palikir", Offset: "+11:00", Region: "Pacific", CountryCode: "FM"},

		// Африка
		{ID: "Africa/Cairo", Name: "Cairo", Offset: "+02:00", Region: "Africa", CountryCode: "EG"},
		{ID: "Africa/Johannesburg", Name: "Johannesburg", Offset: "+02:00", Region: "Africa", CountryCode: "ZA"},
		{ID: "Africa/Lagos", Name: "Lagos", Offset: "+01:00", Region: "Africa", CountryCode: "NG"},
		{ID: "Africa/Nairobi", Name: "Nairobi", Offset: "+03:00", Region: "Africa", CountryCode: "KE"},
		{ID: "Africa/Casablanca", Name: "Casablanca", Offset: "+00:00", Region: "Africa", CountryCode: "MA"},
		{ID: "Africa/Pretoria", Name: "Pretoria", Offset: "+02:00", Region: "Africa", CountryCode: "ZA"},
		{ID: "Africa/Addis_Ababa", Name: "Addis Ababa", Offset: "+03:00", Region: "Africa", CountryCode: "ET"},
		{ID: "Africa/Algiers", Name: "Algiers", Offset: "+01:00", Region: "Africa", CountryCode: "DZ"},
		{ID: "Africa/Luanda", Name: "Luanda", Offset: "+01:00", Region: "Africa", CountryCode: "AO"},
		{ID: "Africa/Porto-Novo", Name: "Porto-Novo", Offset: "+01:00", Region: "Africa", CountryCode: "BJ"},
		{ID: "Africa/Gaborone", Name: "Gaborone", Offset: "+02:00", Region: "Africa", CountryCode: "BW"},
		{ID: "Africa/Ouagadougou", Name: "Ouagadougou", Offset: "+00:00", Region: "Africa", CountryCode: "BF"},
		{ID: "Africa/Bujumbura", Name: "Bujumbura", Offset: "+02:00", Region: "Africa", CountryCode: "BI"},
		{ID: "Africa/Yaounde", Name: "Yaounde", Offset: "+01:00", Region: "Africa", CountryCode: "CM"},
		{ID: "Africa/Praia", Name: "Praia", Offset: "-01:00", Region: "Africa", CountryCode: "CV"},
		{ID: "Africa/Bangui", Name: "Bangui", Offset: "+01:00", Region: "Africa", CountryCode: "CF"},
		{ID: "Africa/Ndjamena", Name: "N'Djamena", Offset: "+01:00", Region: "Africa", CountryCode: "TD"},
		{ID: "Africa/Moroni", Name: "Moroni", Offset: "+03:00", Region: "Africa", CountryCode: "KM"},
		{ID: "Africa/Kinshasa", Name: "Kinshasa", Offset: "+01:00", Region: "Africa", CountryCode: "CD"},
		{ID: "Africa/Brazzaville", Name: "Brazzaville", Offset: "+01:00", Region: "Africa", CountryCode: "CG"},
		{ID: "Africa/Djibouti", Name: "Djibouti", Offset: "+03:00", Region: "Africa", CountryCode: "DJ"},
		{ID: "Africa/Asmara", Name: "Asmara", Offset: "+03:00", Region: "Africa", CountryCode: "ER"},
		{ID: "Africa/Libreville", Name: "Libreville", Offset: "+01:00", Region: "Africa", CountryCode: "GA"},
		{ID: "Africa/Banjul", Name: "Banjul", Offset: "+00:00", Region: "Africa", CountryCode: "GM"},
		{ID: "Africa/Accra", Name: "Accra", Offset: "+00:00", Region: "Africa", CountryCode: "GH"},
		{ID: "Africa/Conakry", Name: "Conakry", Offset: "+00:00", Region: "Africa", CountryCode: "GN"},
		{ID: "Africa/Bissau", Name: "Bissau", Offset: "+00:00", Region: "Africa", CountryCode: "GW"},
		{ID: "Africa/Maseru", Name: "Maseru", Offset: "+02:00", Region: "Africa", CountryCode: "LS"},
		{ID: "Africa/Monrovia", Name: "Monrovia", Offset: "+00:00", Region: "Africa", CountryCode: "LR"},
		{ID: "Africa/Tripoli", Name: "Tripoli", Offset: "+02:00", Region: "Africa", CountryCode: "LY"},
		{ID: "Africa/Antananarivo", Name: "Antananarivo", Offset: "+03:00", Region: "Africa", CountryCode: "MG"},
		{ID: "Africa/Lilongwe", Name: "Lilongwe", Offset: "+02:00", Region: "Africa", CountryCode: "MW"},
		{ID: "Africa/Bamako", Name: "Bamako", Offset: "+00:00", Region: "Africa", CountryCode: "ML"},
		{ID: "Africa/Nouakchott", Name: "Nouakchott", Offset: "+00:00", Region: "Africa", CountryCode: "MR"},
		{ID: "Africa/Maputo", Name: "Maputo", Offset: "+02:00", Region: "Africa", CountryCode: "MZ"},
		{ID: "Africa/Windhoek", Name: "Windhoek", Offset: "+02:00", Region: "Africa", CountryCode: "NA"},
		{ID: "Africa/Niamey", Name: "Niamey", Offset: "+01:00", Region: "Africa", CountryCode: "NE"},
		{ID: "Africa/Kigali", Name: "Kigali", Offset: "+02:00", Region: "Africa", CountryCode: "RW"},
		{ID: "Africa/Dakar", Name: "Dakar", Offset: "+00:00", Region: "Africa", CountryCode: "SN"},
		{ID: "Africa/Freetown", Name: "Freetown", Offset: "+00:00", Region: "Africa", CountryCode: "SL"},
		{ID: "Africa/Mogadishu", Name: "Mogadishu", Offset: "+03:00", Region: "Africa", CountryCode: "SO"},
		{ID: "Africa/Khartoum", Name: "Khartoum", Offset: "+02:00", Region: "Africa", CountryCode: "SD"},
		{ID: "Africa/Juba", Name: "Juba", Offset: "+02:00", Region: "Africa", CountryCode: "SS"},
		{ID: "Africa/Mbabane", Name: "Mbabane", Offset: "+02:00", Region: "Africa", CountryCode: "SZ"},
		{ID: "Africa/Lome", Name: "Lome", Offset: "+00:00", Region: "Africa", CountryCode: "TG"},
		{ID: "Africa/Tunis", Name: "Tunis", Offset: "+01:00", Region: "Africa", CountryCode: "TN"},
		{ID: "Africa/Kampala", Name: "Kampala", Offset: "+03:00", Region: "Africa", CountryCode: "UG"},
		{ID: "Africa/Lusaka", Name: "Lusaka", Offset: "+02:00", Region: "Africa", CountryCode: "ZM"},
		{ID: "Africa/Harare", Name: "Harare", Offset: "+02:00", Region: "Africa", CountryCode: "ZW"},

		// Южная Азия и Индийский океан
		{ID: "Indian/Maldives", Name: "Male", Offset: "+05:00", Region: "Indian Ocean", CountryCode: "MV"},
		{ID: "Indian/Mauritius", Name: "Port Louis", Offset: "+04:00", Region: "Indian Ocean", CountryCode: "MU"},
		{ID: "Indian/Seychelles", Name: "Victoria", Offset: "+04:00", Region: "Indian Ocean", CountryCode: "SC"},
	}
}

// IsValidTimezone проверяет, существует ли указанный часовой пояс в списке доступных
func IsValidTimezone(timezoneID string) bool {
	for _, tz := range GetAvailableTimezones() {
		if tz.ID == timezoneID {
			return true
		}
	}
	return false
}

// GetTimezoneInfo возвращает информацию о часовом поясе по его ID
// Если часовой пояс не найден, возвращает UTC
func GetTimezoneInfo(timezoneID string) TimezoneInfo {
	for _, tz := range GetAvailableTimezones() {
		if tz.ID == timezoneID {
			return tz
		}
	}
	// Если часовой пояс не найден, возвращаем UTC
	return TimezoneInfo{
		ID:          "UTC",
		Name:        "UTC",
		Offset:      "+00:00",
		Region:      "Universal",
		CountryCode: "UN",
	}
}

// GetTimezoneString возвращает строковое представление часового пояса в формате "Name (UTCOffset)"
func GetTimezoneString(timezoneID string) string {
	tz := GetTimezoneInfo(timezoneID)
	return tz.Name + " (" + GetUTCOffset(timezoneID) + ")"
}

// GetUTCOffset возвращает смещение часового пояса в формате "UTC+XX" или "UTC-XX"
func GetUTCOffset(timezoneID string) string {
	tz := GetTimezoneInfo(timezoneID)
	offset := tz.Offset

	// Корректная обработка разных форматов смещения
	var sign string
	var hours, minutes string

	if len(offset) >= 6 { // Формат "+03:00" или "-03:00"
		sign = offset[:1]    // "+" или "-"
		hours = offset[1:3]  // "03"
		minutes = offset[4:] // "00"

		// Убираем лидирующие нули для часов (кроме "00")
		if hours != "00" && hours[0] == '0' {
			hours = hours[1:]
		}

		// Добавляем минуты только если они не "00"
		if minutes != "00" {
			return "UTC" + sign + hours + ":" + minutes
		} else {
			return "UTC" + sign + hours
		}
	}

	// Если формат неизвестен, возвращаем как есть
	return "UTC" + offset
}
