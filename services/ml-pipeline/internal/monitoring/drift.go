package monitoring

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"

	"../../internal/config"
	"../../internal/database"
)

// NewDriftDetector creates a new drift detector
func NewDriftDetector(cfg *config.Config, repos *database.Repositories, logger *zap.Logger) *DriftDetector {
	detector := &DriftDetector{
		config:  cfg,
		logger:  logger,
		repos:   repos,
		methods: make(map[string]DriftDetectionMethod),
	}

	// Register drift detection methods
	detector.methods["ks"] = NewKolmogorovSmirnovDetector(cfg.ML.ModelMonitoring.DriftDetection.DriftThreshold)
	detector.methods["psi"] = NewPSIDetector(cfg.ML.ModelMonitoring.DriftDetection.DriftThreshold)
	detector.methods["jensen_shannon"] = NewJensenShannonDetector(cfg.ML.ModelMonitoring.DriftDetection.DriftThreshold)

	return detector
}

// DetectDrift performs drift detection using the configured method
func (d *DriftDetector) DetectDrift(ctx context.Context, featureName string, reference, current []float64) (*DriftResult, error) {
	method := d.config.ML.ModelMonitoring.DriftDetection.DriftMethod
	
	detector, exists := d.methods[method]
	if !exists {
		return nil, fmt.Errorf("unknown drift detection method: %s", method)
	}

	d.logger.Debug("Detecting drift",
		zap.String("feature", featureName),
		zap.String("method", method),
		zap.Int("reference_size", len(reference)),
		zap.Int("current_size", len(current)))

	return detector.DetectDrift(ctx, reference, current)
}

// KolmogorovSmirnovDetector implements K-S test for drift detection
type KolmogorovSmirnovDetector struct {
	threshold float64
}

// NewKolmogorovSmirnovDetector creates a new K-S detector
func NewKolmogorovSmirnovDetector(threshold float64) *KolmogorovSmirnovDetector {
	return &KolmogorovSmirnovDetector{threshold: threshold}
}

// DetectDrift performs Kolmogorov-Smirnov test
func (d *KolmogorovSmirnovDetector) DetectDrift(ctx context.Context, reference, current []float64) (*DriftResult, error) {
	if len(reference) == 0 || len(current) == 0 {
		return nil, fmt.Errorf("empty datasets provided")
	}

	// Compute K-S statistic
	ksStatistic := d.computeKSStatistic(reference, current)
	
	// For simplified implementation, use the statistic directly
	// In production, you would compute the proper p-value
	isDrift := ksStatistic > d.threshold

	result := &DriftResult{
		IsDrift:        isDrift,
		DriftScore:     ksStatistic,
		Threshold:      d.threshold,
		Method:         "ks",
		StatisticValue: &ksStatistic,
		Metadata: map[string]interface{}{
			"reference_size": len(reference),
			"current_size":   len(current),
			"ks_statistic":   ksStatistic,
		},
		DetectedAt: time.Now(),
	}

	return result, nil
}

// computeKSStatistic computes the Kolmogorov-Smirnov statistic
func (d *KolmogorovSmirnovDetector) computeKSStatistic(reference, current []float64) float64 {
	// Sort both datasets
	refSorted := make([]float64, len(reference))
	copy(refSorted, reference)
	sort.Float64s(refSorted)

	currSorted := make([]float64, len(current))
	copy(currSorted, current)
	sort.Float64s(currSorted)

	// Compute empirical CDFs and find maximum difference
	maxDiff := 0.0
	i, j := 0, 0
	
	for i < len(refSorted) && j < len(currSorted) {
		refCDF := float64(i+1) / float64(len(refSorted))
		currCDF := float64(j+1) / float64(len(currSorted))
		
		diff := math.Abs(refCDF - currCDF)
		if diff > maxDiff {
			maxDiff = diff
		}
		
		if refSorted[i] <= currSorted[j] {
			i++
		} else {
			j++
		}
	}

	return maxDiff
}

func (d *KolmogorovSmirnovDetector) GetMethodName() string {
	return "ks"
}

func (d *KolmogorovSmirnovDetector) GetThreshold() float64 {
	return d.threshold
}

// PSIDetector implements Population Stability Index for drift detection
type PSIDetector struct {
	threshold float64
	numBins   int
}

// NewPSIDetector creates a new PSI detector
func NewPSIDetector(threshold float64) *PSIDetector {
	return &PSIDetector{
		threshold: threshold,
		numBins:   10, // Default number of bins
	}
}

// DetectDrift performs PSI drift detection
func (d *PSIDetector) DetectDrift(ctx context.Context, reference, current []float64) (*DriftResult, error) {
	if len(reference) == 0 || len(current) == 0 {
		return nil, fmt.Errorf("empty datasets provided")
	}

	// Compute PSI
	psi := d.computePSI(reference, current)
	isDrift := psi > d.threshold

	result := &DriftResult{
		IsDrift:        isDrift,
		DriftScore:     psi,
		Threshold:      d.threshold,
		Method:         "psi",
		StatisticValue: &psi,
		Metadata: map[string]interface{}{
			"reference_size": len(reference),
			"current_size":   len(current),
			"psi_score":      psi,
			"num_bins":       d.numBins,
		},
		DetectedAt: time.Now(),
	}

	return result, nil
}

// computePSI computes the Population Stability Index
func (d *PSIDetector) computePSI(reference, current []float64) float64 {
	// Create bins based on reference data quantiles
	refSorted := make([]float64, len(reference))
	copy(refSorted, reference)
	sort.Float64s(refSorted)

	// Create bin boundaries
	binBoundaries := make([]float64, d.numBins+1)
	for i := 0; i <= d.numBins; i++ {
		if i == 0 {
			binBoundaries[i] = refSorted[0] - 1e-6
		} else if i == d.numBins {
			binBoundaries[i] = refSorted[len(refSorted)-1] + 1e-6
		} else {
			idx := int(float64(len(refSorted)) * float64(i) / float64(d.numBins))
			if idx >= len(refSorted) {
				idx = len(refSorted) - 1
			}
			binBoundaries[i] = refSorted[idx]
		}
	}

	// Count frequencies in each bin
	refCounts := d.countFrequencies(reference, binBoundaries)
	currCounts := d.countFrequencies(current, binBoundaries)

	// Convert to proportions
	refProportions := make([]float64, len(refCounts))
	currProportions := make([]float64, len(currCounts))

	for i := range refCounts {
		refProportions[i] = float64(refCounts[i]) / float64(len(reference))
		currProportions[i] = float64(currCounts[i]) / float64(len(current))
	}

	// Compute PSI
	psi := 0.0
	for i := range refProportions {
		if refProportions[i] > 0 && currProportions[i] > 0 {
			psi += (currProportions[i] - refProportions[i]) * math.Log(currProportions[i]/refProportions[i])
		}
	}

	return psi
}

// countFrequencies counts frequencies in bins
func (d *PSIDetector) countFrequencies(data []float64, boundaries []float64) []int {
	counts := make([]int, len(boundaries)-1)
	
	for _, value := range data {
		for i := 0; i < len(boundaries)-1; i++ {
			if value >= boundaries[i] && value < boundaries[i+1] {
				counts[i]++
				break
			}
		}
	}
	
	return counts
}

func (d *PSIDetector) GetMethodName() string {
	return "psi"
}

func (d *PSIDetector) GetThreshold() float64 {
	return d.threshold
}

// JensenShannonDetector implements Jensen-Shannon divergence for drift detection
type JensenShannonDetector struct {
	threshold float64
	numBins   int
}

// NewJensenShannonDetector creates a new Jensen-Shannon detector
func NewJensenShannonDetector(threshold float64) *JensenShannonDetector {
	return &JensenShannonDetector{
		threshold: threshold,
		numBins:   20,
	}
}

// DetectDrift performs Jensen-Shannon divergence drift detection
func (d *JensenShannonDetector) DetectDrift(ctx context.Context, reference, current []float64) (*DriftResult, error) {
	if len(reference) == 0 || len(current) == 0 {
		return nil, fmt.Errorf("empty datasets provided")
	}

	// Compute Jensen-Shannon divergence
	js := d.computeJensenShannonDivergence(reference, current)
	isDrift := js > d.threshold

	result := &DriftResult{
		IsDrift:        isDrift,
		DriftScore:     js,
		Threshold:      d.threshold,
		Method:         "jensen_shannon",
		StatisticValue: &js,
		Metadata: map[string]interface{}{
			"reference_size": len(reference),
			"current_size":   len(current),
			"js_divergence":  js,
			"num_bins":       d.numBins,
		},
		DetectedAt: time.Now(),
	}

	return result, nil
}

// computeJensenShannonDivergence computes the Jensen-Shannon divergence
func (d *JensenShannonDetector) computeJensenShannonDivergence(reference, current []float64) float64 {
	// Create histograms
	refHist := d.createHistogram(reference)
	currHist := d.createHistogram(current)

	// Normalize to probabilities
	refProbs := d.normalizeProbabilities(refHist)
	currProbs := d.normalizeProbabilities(currHist)

	// Compute Jensen-Shannon divergence
	return d.jensenShannonDivergence(refProbs, currProbs)
}

// createHistogram creates a histogram with fixed number of bins
func (d *JensenShannonDetector) createHistogram(data []float64) []float64 {
	if len(data) == 0 {
		return make([]float64, d.numBins)
	}

	min, max := stat.Bounds(data, nil)
	if min == max {
		// All values are the same
		hist := make([]float64, d.numBins)
		hist[0] = float64(len(data))
		return hist
	}

	binWidth := (max - min) / float64(d.numBins)
	hist := make([]float64, d.numBins)

	for _, value := range data {
		binIndex := int((value - min) / binWidth)
		if binIndex >= d.numBins {
			binIndex = d.numBins - 1
		}
		hist[binIndex]++
	}

	return hist
}

// normalizeProbabilities normalizes histogram to probabilities
func (d *JensenShannonDetector) normalizeProbabilities(hist []float64) []float64 {
	total := 0.0
	for _, count := range hist {
		total += count
	}

	if total == 0 {
		return make([]float64, len(hist))
	}

	probs := make([]float64, len(hist))
	for i, count := range hist {
		probs[i] = count / total
		// Add small epsilon to avoid log(0)
		if probs[i] == 0 {
			probs[i] = 1e-10
		}
	}

	return probs
}

// jensenShannonDivergence computes the Jensen-Shannon divergence
func (d *JensenShannonDetector) jensenShannonDivergence(p, q []float64) float64 {
	if len(p) != len(q) {
		return 0.0
	}

	// Compute average distribution M = (P + Q) / 2
	m := make([]float64, len(p))
	for i := range m {
		m[i] = (p[i] + q[i]) / 2.0
	}

	// Compute KL divergences
	klPM := d.klDivergence(p, m)
	klQM := d.klDivergence(q, m)

	// Jensen-Shannon divergence
	return 0.5*klPM + 0.5*klQM
}

// klDivergence computes the Kullback-Leibler divergence
func (d *JensenShannonDetector) klDivergence(p, q []float64) float64 {
	kl := 0.0
	for i := range p {
		if p[i] > 0 && q[i] > 0 {
			kl += p[i] * math.Log(p[i]/q[i])
		}
	}
	return kl
}

func (d *JensenShannonDetector) GetMethodName() string {
	return "jensen_shannon"
}

func (d *JensenShannonDetector) GetThreshold() float64 {
	return d.threshold
}