package standardization

import (
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/bbalet/stopwords"
	"github.com/kljensen/snowball"
)

// Engine handles data standardization for entity resolution
type Engine struct {
	logger *slog.Logger
}

// StandardizedName represents a standardized name with metadata
type StandardizedName struct {
	Original     string   `json:"original"`
	Standardized string   `json:"standardized"`
	Tokens       []string `json:"tokens"`
	Phonetic     string   `json:"phonetic"`
	Metaphone    string   `json:"metaphone"`
}

// StandardizedAddress represents a standardized address
type StandardizedAddress struct {
	Original       string `json:"original"`
	Standardized   string `json:"standardized"`
	StreetNumber   string `json:"street_number"`
	StreetName     string `json:"street_name"`
	StreetType     string `json:"street_type"`
	City           string `json:"city"`
	State          string `json:"state"`
	PostalCode     string `json:"postal_code"`
	Country        string `json:"country"`
}

// StandardizedPhone represents a standardized phone number
type StandardizedPhone struct {
	Original      string `json:"original"`
	Standardized  string `json:"standardized"`
	CountryCode   string `json:"country_code"`
	AreaCode      string `json:"area_code"`
	Number        string `json:"number"`
	Extension     string `json:"extension"`
}

// NewEngine creates a new standardization engine
func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		logger: logger,
	}
}

// StandardizeName standardizes a person or organization name
func (e *Engine) StandardizeName(name string) *StandardizedName {
	if name == "" {
		return &StandardizedName{
			Original:     name,
			Standardized: "",
			Tokens:       []string{},
			Phonetic:     "",
			Metaphone:    "",
		}
	}

	// Clean and normalize the name
	cleaned := e.cleanName(name)
	tokens := e.tokenizeName(cleaned)
	standardized := e.standardizeNameTokens(tokens)
	phonetic := e.generatePhonetic(standardized)
	metaphone := e.generateMetaphone(standardized)

	return &StandardizedName{
		Original:     name,
		Standardized: standardized,
		Tokens:       tokens,
		Phonetic:     phonetic,
		Metaphone:    metaphone,
	}
}

// StandardizeAddress standardizes an address
func (e *Engine) StandardizeAddress(address string) *StandardizedAddress {
	if address == "" {
		return &StandardizedAddress{
			Original:     address,
			Standardized: "",
		}
	}

	// Clean and normalize the address
	cleaned := e.cleanAddress(address)
	components := e.parseAddressComponents(cleaned)
	standardized := e.standardizeAddressComponents(components)

	return &StandardizedAddress{
		Original:       address,
		Standardized:   standardized.String(),
		StreetNumber:   standardized.StreetNumber,
		StreetName:     standardized.StreetName,
		StreetType:     standardized.StreetType,
		City:           standardized.City,
		State:          standardized.State,
		PostalCode:     standardized.PostalCode,
		Country:        standardized.Country,
	}
}

// StandardizePhone standardizes a phone number
func (e *Engine) StandardizePhone(phone string) *StandardizedPhone {
	if phone == "" {
		return &StandardizedPhone{
			Original:     phone,
			Standardized: "",
		}
	}

	// Clean and parse the phone number
	cleaned := e.cleanPhone(phone)
	components := e.parsePhoneComponents(cleaned)
	standardized := e.formatStandardizedPhone(components)

	return &StandardizedPhone{
		Original:      phone,
		Standardized:  standardized,
		CountryCode:   components.CountryCode,
		AreaCode:      components.AreaCode,
		Number:        components.Number,
		Extension:     components.Extension,
	}
}

// StandardizeEmail standardizes an email address
func (e *Engine) StandardizeEmail(email string) string {
	if email == "" {
		return ""
	}

	// Convert to lowercase and trim
	email = strings.ToLower(strings.TrimSpace(email))

	// Handle Gmail-specific normalization (remove dots and plus aliases)
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		localPart := parts[0]
		domain := parts[1]

		if domain == "gmail.com" || domain == "googlemail.com" {
			// Remove dots from local part
			localPart = strings.ReplaceAll(localPart, ".", "")
			// Remove everything after +
			if plusIndex := strings.Index(localPart, "+"); plusIndex != -1 {
				localPart = localPart[:plusIndex]
			}
			domain = "gmail.com" // Normalize googlemail.com to gmail.com
		}

		return localPart + "@" + domain
	}

	return email
}

// Name standardization helpers

func (e *Engine) cleanName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove extra whitespace
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	name = strings.TrimSpace(name)

	// Remove punctuation except hyphens and apostrophes
	name = regexp.MustCompile(`[^\w\s\-']`).ReplaceAllString(name, "")

	return name
}

func (e *Engine) tokenizeName(name string) []string {
	// Split by whitespace and remove empty tokens
	tokens := strings.Fields(name)
	
	// Remove common stop words and titles
	var filtered []string
	for _, token := range tokens {
		if !e.isStopWord(token) && !e.isTitle(token) {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

func (e *Engine) standardizeNameTokens(tokens []string) string {
	var standardized []string
	
	for _, token := range tokens {
		// Apply stemming
		stemmed, err := snowball.Stem(token, "english", true)
		if err == nil {
			token = stemmed
		}
		
		// Normalize common name variations
		token = e.normalizeNameToken(token)
		
		if token != "" {
			standardized = append(standardized, token)
		}
	}

	return strings.Join(standardized, " ")
}

func (e *Engine) isStopWord(word string) bool {
	return stopwords.IsStopWord(word, "en")
}

func (e *Engine) isTitle(word string) bool {
	titles := map[string]bool{
		"mr": true, "mrs": true, "ms": true, "miss": true, "dr": true,
		"prof": true, "sir": true, "madam": true, "lord": true, "lady": true,
		"jr": true, "sr": true, "ii": true, "iii": true, "iv": true,
	}
	return titles[strings.ToLower(word)]
}

func (e *Engine) normalizeNameToken(token string) string {
	// Common name normalizations
	normalizations := map[string]string{
		"william":   "bill",
		"robert":    "bob",
		"richard":   "dick",
		"michael":   "mike",
		"christopher": "chris",
		"matthew":   "matt",
		"andrew":    "andy",
		"anthony":   "tony",
		"joseph":    "joe",
		"david":     "dave",
		"james":     "jim",
		"john":      "jack",
		"corporation": "corp",
		"company":   "co",
		"incorporated": "inc",
		"limited":   "ltd",
		"llc":       "llc",
		"lp":        "lp",
	}

	if normalized, exists := normalizations[token]; exists {
		return normalized
	}

	return token
}

// Phonetic encoding
func (e *Engine) generatePhonetic(text string) string {
	// Simple phonetic encoding (Soundex-like)
	if text == "" {
		return ""
	}

	text = strings.ToUpper(text)
	text = regexp.MustCompile(`[^A-Z]`).ReplaceAllString(text, "")

	if len(text) == 0 {
		return ""
	}

	// Keep first letter
	result := string(text[0])

	// Replace consonants with digits
	replacements := map[rune]rune{
		'B': '1', 'F': '1', 'P': '1', 'V': '1',
		'C': '2', 'G': '2', 'J': '2', 'K': '2', 'Q': '2', 'S': '2', 'X': '2', 'Z': '2',
		'D': '3', 'T': '3',
		'L': '4',
		'M': '5', 'N': '5',
		'R': '6',
	}

	for _, char := range text[1:] {
		if replacement, exists := replacements[char]; exists {
			result += string(replacement)
		}
	}

	// Remove duplicates and pad/truncate to 4 characters
	result = e.removeDuplicateChars(result)
	for len(result) < 4 {
		result += "0"
	}
	if len(result) > 4 {
		result = result[:4]
	}

	return result
}

func (e *Engine) generateMetaphone(text string) string {
	// Simplified Double Metaphone algorithm
	if text == "" {
		return ""
	}

	text = strings.ToUpper(text)
	text = regexp.MustCompile(`[^A-Z]`).ReplaceAllString(text, "")

	// Apply basic metaphone rules (simplified)
	result := text
	
	// Replace common patterns
	replacements := [][]string{
		{"PH", "F"},
		{"GH", "F"},
		{"CK", "K"},
		{"SCH", "SK"},
		{"TH", "0"},
		{"SH", "X"},
		{"CH", "X"},
	}

	for _, replacement := range replacements {
		result = strings.ReplaceAll(result, replacement[0], replacement[1])
	}

	// Remove vowels except at the beginning
	if len(result) > 1 {
		vowels := "AEIOU"
		var newResult strings.Builder
		newResult.WriteByte(result[0])
		
		for i := 1; i < len(result); i++ {
			if !strings.ContainsRune(vowels, rune(result[i])) {
				newResult.WriteByte(result[i])
			}
		}
		result = newResult.String()
	}

	// Pad/truncate to 4 characters
	for len(result) < 4 {
		result += "0"
	}
	if len(result) > 4 {
		result = result[:4]
	}

	return result
}

func (e *Engine) removeDuplicateChars(s string) string {
	if len(s) <= 1 {
		return s
	}

	var result strings.Builder
	result.WriteByte(s[0])

	for i := 1; i < len(s); i++ {
		if s[i] != s[i-1] {
			result.WriteByte(s[i])
		}
	}

	return result.String()
}

// Address standardization

type AddressComponents struct {
	StreetNumber string
	StreetName   string
	StreetType   string
	City         string
	State        string
	PostalCode   string
	Country      string
}

func (ac *AddressComponents) String() string {
	var parts []string
	
	if ac.StreetNumber != "" {
		parts = append(parts, ac.StreetNumber)
	}
	if ac.StreetName != "" {
		parts = append(parts, ac.StreetName)
	}
	if ac.StreetType != "" {
		parts = append(parts, ac.StreetType)
	}
	if ac.City != "" {
		parts = append(parts, ac.City)
	}
	if ac.State != "" {
		parts = append(parts, ac.State)
	}
	if ac.PostalCode != "" {
		parts = append(parts, ac.PostalCode)
	}
	if ac.Country != "" {
		parts = append(parts, ac.Country)
	}

	return strings.Join(parts, " ")
}

func (e *Engine) cleanAddress(address string) string {
	// Convert to lowercase and remove extra whitespace
	address = strings.ToLower(address)
	address = regexp.MustCompile(`\s+`).ReplaceAllString(address, " ")
	address = strings.TrimSpace(address)

	// Normalize common abbreviations
	abbreviations := map[string]string{
		" st ":     " street ",
		" ave ":    " avenue ",
		" blvd ":   " boulevard ",
		" rd ":     " road ",
		" dr ":     " drive ",
		" ln ":     " lane ",
		" ct ":     " court ",
		" pl ":     " place ",
		" apt ":    " apartment ",
		" ste ":    " suite ",
		" bldg ":   " building ",
		" fl ":     " floor ",
	}

	for abbrev, full := range abbreviations {
		address = strings.ReplaceAll(address, abbrev, full)
	}

	return address
}

func (e *Engine) parseAddressComponents(address string) *AddressComponents {
	components := &AddressComponents{}

	// Simple regex-based parsing (in production, use a proper address parser)
	
	// Extract postal code (US format)
	postalCodeRegex := regexp.MustCompile(`\b(\d{5}(?:-\d{4})?)\b`)
	if matches := postalCodeRegex.FindStringSubmatch(address); len(matches) > 1 {
		components.PostalCode = matches[1]
		address = postalCodeRegex.ReplaceAllString(address, "")
	}

	// Extract state (2-letter abbreviation)
	stateRegex := regexp.MustCompile(`\b([A-Z]{2})\b`)
	if matches := stateRegex.FindStringSubmatch(strings.ToUpper(address)); len(matches) > 1 {
		components.State = strings.ToLower(matches[1])
		address = strings.ReplaceAll(strings.ToLower(address), strings.ToLower(matches[1]), "")
	}

	// Extract street number (at the beginning)
	streetNumberRegex := regexp.MustCompile(`^\s*(\d+[a-z]?)\s+`)
	if matches := streetNumberRegex.FindStringSubmatch(address); len(matches) > 1 {
		components.StreetNumber = matches[1]
		address = streetNumberRegex.ReplaceAllString(address, "")
	}

	// Extract street type (at the end of remaining address)
	streetTypes := []string{"street", "avenue", "boulevard", "road", "drive", "lane", "court", "place"}
	for _, streetType := range streetTypes {
		pattern := regexp.MustCompile(`\b` + streetType + `\b`)
		if pattern.MatchString(address) {
			components.StreetType = streetType
			address = pattern.ReplaceAllString(address, "")
			break
		}
	}

	// Remaining parts are street name and city
	parts := strings.Fields(strings.TrimSpace(address))
	if len(parts) > 0 {
		// Assume last part(s) are city, first part(s) are street name
		if len(parts) > 2 {
			components.StreetName = strings.Join(parts[:len(parts)-1], " ")
			components.City = parts[len(parts)-1]
		} else if len(parts) == 2 {
			components.StreetName = parts[0]
			components.City = parts[1]
		} else {
			components.StreetName = parts[0]
		}
	}

	return components
}

func (e *Engine) standardizeAddressComponents(components *AddressComponents) *AddressComponents {
	standardized := &AddressComponents{
		StreetNumber: components.StreetNumber,
		StreetName:   e.standardizeStreetName(components.StreetName),
		StreetType:   e.standardizeStreetType(components.StreetType),
		City:         e.standardizeCity(components.City),
		State:        e.standardizeState(components.State),
		PostalCode:   components.PostalCode,
		Country:      e.standardizeCountry(components.Country),
	}

	return standardized
}

func (e *Engine) standardizeStreetName(streetName string) string {
	if streetName == "" {
		return ""
	}

	// Remove common prefixes/suffixes and normalize
	streetName = strings.TrimSpace(streetName)
	
	// Convert ordinals
	ordinalRegex := regexp.MustCompile(`\b(\d+)(st|nd|rd|th)\b`)
	streetName = ordinalRegex.ReplaceAllString(streetName, "$1")

	return streetName
}

func (e *Engine) standardizeStreetType(streetType string) string {
	// Standardize street type abbreviations
	standardTypes := map[string]string{
		"st":     "street",
		"ave":    "avenue",
		"blvd":   "boulevard",
		"rd":     "road",
		"dr":     "drive",
		"ln":     "lane",
		"ct":     "court",
		"pl":     "place",
		"cir":    "circle",
		"way":    "way",
		"pkwy":   "parkway",
		"hwy":    "highway",
	}

	if standard, exists := standardTypes[streetType]; exists {
		return standard
	}

	return streetType
}

func (e *Engine) standardizeCity(city string) string {
	if city == "" {
		return ""
	}

	// Basic normalization
	city = strings.TrimSpace(city)
	
	// Capitalize first letter of each word
	words := strings.Fields(city)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

func (e *Engine) standardizeState(state string) string {
	if state == "" {
		return ""
	}

	// Convert to uppercase
	state = strings.ToUpper(strings.TrimSpace(state))

	// Map state names to abbreviations
	stateMap := map[string]string{
		"ALABAMA":        "AL",
		"ALASKA":         "AK",
		"ARIZONA":        "AZ",
		"ARKANSAS":       "AR",
		"CALIFORNIA":     "CA",
		"COLORADO":       "CO",
		"CONNECTICUT":    "CT",
		"DELAWARE":       "DE",
		"FLORIDA":        "FL",
		"GEORGIA":        "GA",
		// Add more state mappings as needed
	}

	if abbrev, exists := stateMap[state]; exists {
		return abbrev
	}

	// If already an abbreviation, return as is
	if len(state) == 2 {
		return state
	}

	return state
}

func (e *Engine) standardizeCountry(country string) string {
	if country == "" {
		return "US" // Default to US
	}

	country = strings.ToUpper(strings.TrimSpace(country))

	// Map country names to ISO codes
	countryMap := map[string]string{
		"UNITED STATES":         "US",
		"USA":                   "US",
		"AMERICA":               "US",
		"CANADA":                "CA",
		"UNITED KINGDOM":        "GB",
		"UK":                    "GB",
		"GREAT BRITAIN":         "GB",
		"ENGLAND":               "GB",
	}

	if code, exists := countryMap[country]; exists {
		return code
	}

	return country
}

// Phone standardization

type PhoneComponents struct {
	CountryCode string
	AreaCode    string
	Number      string
	Extension   string
}

func (e *Engine) cleanPhone(phone string) string {
	// Remove all non-numeric characters except 'x' for extension
	phone = regexp.MustCompile(`[^\dx]+`).ReplaceAllString(phone, "")
	return strings.TrimSpace(phone)
}

func (e *Engine) parsePhoneComponents(phone string) *PhoneComponents {
	components := &PhoneComponents{}

	// Handle extension
	if extIndex := strings.Index(phone, "x"); extIndex != -1 {
		components.Extension = phone[extIndex+1:]
		phone = phone[:extIndex]
	}

	// Remove any remaining non-digits
	phone = regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Parse based on length
	switch len(phone) {
	case 10:
		// US number without country code
		components.CountryCode = "1"
		components.AreaCode = phone[:3]
		components.Number = phone[3:]
	case 11:
		// US number with country code
		if phone[0] == '1' {
			components.CountryCode = "1"
			components.AreaCode = phone[1:4]
			components.Number = phone[4:]
		} else {
			components.Number = phone
		}
	case 7:
		// Local number without area code
		components.Number = phone
	default:
		// Other formats
		components.Number = phone
	}

	return components
}

func (e *Engine) formatStandardizedPhone(components *PhoneComponents) string {
	if components.Number == "" {
		return ""
	}

	var parts []string

	if components.CountryCode != "" {
		parts = append(parts, "+"+components.CountryCode)
	}

	if components.AreaCode != "" {
		parts = append(parts, "("+components.AreaCode+")")
	}

	// Format number with dash
	if len(components.Number) == 7 {
		parts = append(parts, components.Number[:3]+"-"+components.Number[3:])
	} else {
		parts = append(parts, components.Number)
	}

	if components.Extension != "" {
		parts = append(parts, "x"+components.Extension)
	}

	return strings.Join(parts, " ")
}