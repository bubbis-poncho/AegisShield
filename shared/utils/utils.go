// Shared Utilities - T026
// Constitutional Principle: Code Quality & Modular Code

package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// ID Generation Utilities
func GenerateID() string {
	return uuid.New().String()
}

func GenerateShortID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
}

func GenerateSecureID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Hash Utilities
func HashString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func HashWithSalt(input, salt string) string {
	combined := input + salt
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// String Utilities
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

func IsNotEmpty(s string) bool {
	return !IsEmpty(s)
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsIgnoreCase(slice []string, item string) bool {
	itemLower := strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == itemLower {
			return true
		}
	}
	return false
}

func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

func SanitizeString(s string) string {
	// Remove control characters and trim whitespace
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(cleaned)
}

// Validation Utilities
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func IsValidPhoneNumber(phone string) bool {
	// Basic phone number validation (supports international formats)
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	cleaned := regexp.MustCompile(`[^\d+]`).ReplaceAllString(phone, "")
	return phoneRegex.MatchString(cleaned)
}

func IsValidCountryCode(code string) bool {
	// ISO 3166-1 alpha-2 country codes
	validCodes := map[string]bool{
		"US": true, "CA": true, "GB": true, "DE": true, "FR": true,
		"IT": true, "ES": true, "NL": true, "BE": true, "CH": true,
		"AU": true, "NZ": true, "JP": true, "CN": true, "IN": true,
		"BR": true, "MX": true, "AR": true, "CL": true, "PE": true,
		"SG": true, "HK": true, "AE": true, "SA": true, "IL": true,
		// Add more as needed
	}
	return validCodes[strings.ToUpper(code)]
}

func IsValidCurrencyCode(code string) bool {
	// ISO 4217 currency codes
	validCodes := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true, "CAD": true,
		"AUD": true, "CHF": true, "CNY": true, "INR": true, "BRL": true,
		"MXN": true, "SGD": true, "HKD": true, "AED": true, "SAR": true,
		"ILS": true, "ZAR": true, "RUB": true, "KRW": true, "SEK": true,
		// Add more as needed
	}
	return validCodes[strings.ToUpper(code)]
}

func IsValidAmount(amount float64) bool {
	return amount >= 0 && amount <= 999999999.99
}

// Time Utilities
func TimeToString(t time.Time) string {
	return t.Format(time.RFC3339)
}

func StringToTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func IsWithinTimeRange(t, start, end time.Time) bool {
	return t.After(start) && t.Before(end)
}

func DaysBetween(start, end time.Time) int {
	duration := end.Sub(start)
	return int(duration.Hours() / 24)
}

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

func IsBusinessDay(t time.Time) bool {
	weekday := t.Weekday()
	return weekday != time.Saturday && weekday != time.Sunday
}

// JSON Utilities
func ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

func ToJSONIndented(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Map Utilities
func MergeStringMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func CopyStringMap(original map[string]string) map[string]string {
	copy := make(map[string]string)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func MapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func MapValues(m map[string]string) []string {
	values := make([]string, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// Numeric Utilities
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func FormatFloat(f float64, precision int) string {
	return fmt.Sprintf("%."+strconv.Itoa(precision)+"f", f)
}

func RoundToDecimals(f float64, decimals int) float64 {
	multiplier := float64(1)
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	return float64(int(f*multiplier+0.5)) / multiplier
}

func MinFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func MaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ClampFloat(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Reflection Utilities
func IsNil(v interface{}) bool {
	if v == nil {
		return true
	}
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return val.IsNil()
	default:
		return false
	}
}

func GetFieldValue(obj interface{}, fieldName string) (interface{}, error) {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not a struct")
	}
	
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}
	
	return field.Interface(), nil
}

func SetFieldValue(obj interface{}, fieldName string, value interface{}) error {
	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("object must be a pointer to struct")
	}
	
	val = val.Elem()
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", fieldName)
	}
	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", fieldName)
	}
	
	fieldValue := reflect.ValueOf(value)
	if field.Type() != fieldValue.Type() {
		return fmt.Errorf("type mismatch for field %s", fieldName)
	}
	
	field.Set(fieldValue)
	return nil
}

// Error Utilities
type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	var messages []string
	for _, err := range m.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

func NewMultiError() *MultiError {
	return &MultiError{Errors: make([]error, 0)}
}

// Retry Utilities
type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
	Backoff     func(attempt int, delay time.Duration) time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
		Backoff: func(attempt int, delay time.Duration) time.Duration {
			return delay * time.Duration(attempt)
		},
	}
}

func Retry(fn func() error, config RetryConfig) error {
	var lastErr error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		
		if attempt < config.MaxAttempts {
			delay := config.Backoff(attempt, config.Delay)
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// Risk Score Utilities
func CalculateRiskScore(factors []float64, weights []float64) float64 {
	if len(factors) != len(weights) {
		return 0.0
	}
	
	var weightedSum, totalWeight float64
	for i, factor := range factors {
		weightedSum += factor * weights[i]
		totalWeight += weights[i]
	}
	
	if totalWeight == 0 {
		return 0.0
	}
	
	return ClampFloat(weightedSum/totalWeight, 0.0, 100.0)
}

func NormalizeScore(score, min, max float64) float64 {
	if max == min {
		return 0.0
	}
	return ClampFloat((score-min)/(max-min)*100, 0.0, 100.0)
}

func ScoreToRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "CRITICAL"
	case score >= 60:
		return "HIGH"
	case score >= 40:
		return "MEDIUM"
	case score >= 20:
		return "LOW"
	default:
		return "MINIMAL"
	}
}

// Batch Processing Utilities
func BatchProcess[T any](items []T, batchSize int, processor func([]T) error) error {
	if batchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		
		batch := items[i:end]
		if err := processor(batch); err != nil {
			return fmt.Errorf("batch processing failed at index %d: %w", i, err)
		}
	}
	
	return nil
}

// Geographic Utilities
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0 // Earth's radius in kilometers
	
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadius * c
}

func IsHighRiskCountry(countryCode string) bool {
	highRiskCountries := map[string]bool{
		"AF": true, // Afghanistan
		"BY": true, // Belarus
		"MM": true, // Myanmar
		"KP": true, // North Korea
		"IR": true, // Iran
		"SY": true, // Syria
		// Add more as needed based on sanctions lists
	}
	return highRiskCountries[strings.ToUpper(countryCode)]
}

// Cache Key Utilities
func BuildCacheKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func HashCacheKey(key string) string {
	if len(key) <= 250 { // Redis key length limit
		return key
	}
	return HashString(key)
}

// Configuration Utilities
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("environment variable %s is required", key))
	}
	return value
}

// HTTP Utilities
func ExtractIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func GetUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func GetRequestID(r *http.Request) string {
	// Check for existing request ID in headers
	if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
		return reqID
	}
	if reqID := r.Header.Get("Request-ID"); reqID != "" {
		return reqID
	}
	// Generate new request ID
	return GenerateID()
}