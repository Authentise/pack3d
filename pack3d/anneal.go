package pack3d

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/fogleman/fauxgl"
)

type AnnealCallback func(Annealable)

type Annealable interface {
	Energy() float64
	DoMove([]fauxgl.Vector, fauxgl.Vector, int) (Undo, int)
	UndoMove(Undo)
	Copy() Annealable
}

// Step-size scaling: when DoMove often needs many attempts to find a valid move
// (dense packing, shallow minima), we temporarily increase Deviation to help escape.
const (
	rejectWindowSize        = 200  // steps between scaling decisions
	highRejectThreshold     = 50   // DoMove attempts before we count it as "struggling"
	scaleUpRejectRatio      = 0.3  // scale up Deviation if >= 30% of moves struggle
	scaleDownRejectRatio    = 0.1  // scale down when <= 10% struggle
	deviationScaleFactor    = 1.5  // multiply/divide factor when adjusting
	deviationMaxMultiplier  = 4.0  // cap Deviation at base * this
)

const (
	progressThrottleSeconds = 5.0  // min time between progress prints
	progressMaxPrints       = 10   // max prints per run (excl. final)
)

func Anneal(state Annealable, maxTemp, minTemp float64, steps int, callback AnnealCallback, singleStlSize []fauxgl.Vector, frameSize fauxgl.Vector, packItemNum int) (Annealable, int) {
	start := time.Now()
	factor := -math.Log(maxTemp / minTemp)
	state = state.Copy()
	bestState := state.Copy()
	if callback != nil {
		callback(bestState)
	}
	bestEnergy := state.Energy()
	previousEnergy := bestEnergy
	progressInterval := steps / progressMaxPrints
	var cycleIndex int
	var lastProgressTime float64

	// Track rejection rate to scale step size when stuck in dense packings.
	var baseDeviation float64
	var highRejectCount, windowCount int
	if model, ok := state.(*Model); ok {
		baseDeviation = model.Deviation // remember initial step size for scale-down
	}

	for step := 0; step < steps; step++ {
		pct := float64(step) / float64(steps-1)
		temp := maxTemp * math.Exp(factor*pct)
		if step%progressInterval == 0 {
			elapsed := time.Since(start).Seconds()
			if elapsed >= lastProgressTime+progressThrottleSeconds {
				showProgress(step, steps, bestEnergy, elapsed)
				lastProgressTime = elapsed
			}
		}
		undo, ntime := state.DoMove(singleStlSize, frameSize, packItemNum)
		cycleIndex = ntime
		if ntime >= 100{
			return bestState, ntime
		}

		// Scale step size when reject rate is high. Dense packings cause DoMove to
		// reject most proposals (intersection, containment, out-of-bounds), so only
		// tiny moves succeed; we then creep slowly out of minima. Boosting Deviation
		// allows larger translation steps to be proposed and accepted.
		if model, ok := state.(*Model); ok && baseDeviation > 0 {
			windowCount++
			if ntime >= highRejectThreshold {
				highRejectCount++
			}
			if windowCount >= rejectWindowSize {
				ratio := float64(highRejectCount) / float64(windowCount)
				maxDeviation := baseDeviation * deviationMaxMultiplier
				if ratio >= scaleUpRejectRatio && model.Deviation < maxDeviation {
					model.Deviation = math.Min(model.Deviation*deviationScaleFactor, maxDeviation)
				} else if ratio <= scaleDownRejectRatio && model.Deviation > baseDeviation {
					model.Deviation = math.Max(model.Deviation/deviationScaleFactor, baseDeviation)
				}
				highRejectCount, windowCount = 0, 0
			}
		}
		energy := state.Energy()
		change := energy - previousEnergy
		if change > 0 && math.Exp(-change/temp) < rand.Float64() {
			state.UndoMove(undo)
		} else {
			previousEnergy = energy
			if energy < bestEnergy {
				bestEnergy = energy
				bestState = state.Copy()
				if callback != nil {
					callback(bestState)
				}
			}
		}
	}
	showProgress(steps, steps, bestEnergy, time.Since(start).Seconds())
	fmt.Println()
	return bestState, cycleIndex
}

/* This function shows progress, not necessary*/
func showProgress(i, n int, e, d float64) {
	pct := int(100 * float64(i) / float64(n))
	fmt.Printf("  %3d%% [", pct)
	for p := 0; p < 100; p += 3 {
		if pct > p {
			fmt.Print("=")
		} else {
			fmt.Print(" ")
		}
	}
	fmt.Printf("] %.6f %.3fs    \r", e, d)
}
