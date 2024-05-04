package feral

import (
	"time"

	"github.com/wowsims/cata/sim/core"
)

func (cat *FeralDruid) calcExpectedSwipeDamage(sim *core.Simulation) (float64, float64) {
	expectedSwipeDamage := 0.0
	for _, aoeTarget := range sim.Encounter.TargetUnits {
		expectedSwipeDamage += cat.SwipeCat.ExpectedInitialDamage(sim, aoeTarget)
	}
	swipeDPE := expectedSwipeDamage / cat.SwipeCat.DefaultCast.Cost
	return expectedSwipeDamage, swipeDPE
}

func (cat *FeralDruid) doAoeRotation(sim *core.Simulation) (bool, time.Duration) {
	// Store state variables for re-use
	rotation := &cat.Rotation
	curEnergy := cat.CurrentEnergy()
	curCp := cat.ComboPoints()
	isClearcast := cat.ClearcastingAura.IsActive()
	simTimeRemain := sim.GetRemainingDuration()
	regenRate := cat.EnergyRegenPerSecond()

	// Keep up Sunder debuff if not provided externally
	if rotation.MaintainFaerieFire {
		for _, aoeTarget := range sim.Encounter.TargetUnits {
			if cat.ShouldFaerieFire(sim, aoeTarget) {
				cat.FaerieFire.Cast(sim, aoeTarget)
				return false, 0
			}
		}
	}

	// Roar check
	roarNow := (curCp >= 1) && !cat.SavageRoarAura.IsActive()

	if roarNow {
		// Compare DPE versus Swipe to see if it's worth casting
		baseAutoDamage := cat.MHAutoSpell.ExpectedInitialDamage(sim, cat.CurrentTarget)
		buffEnd := min(sim.Duration, sim.CurrentTime+cat.SavageRoarDurationTable[curCp])
		numBuffedAutos := 1 + int32((buffEnd-cat.AutoAttacks.NextAttackAt())/cat.AutoAttacks.MainhandSwingSpeed())
		roarDPE := (cat.GetSavageRoarMultiplier() - 1) * baseAutoDamage * float64(numBuffedAutos) / cat.SavageRoar.DefaultCast.Cost
		_, swipeDPE := cat.calcExpectedSwipeDamage(sim)

		if sim.Log != nil {
			cat.Log(sim, "Roar DPE = %.1f, Swipe DPE = %.1f", roarDPE, swipeDPE)
		}

		roarNow = (roarDPE >= swipeDPE)
	}

	// Rake check
	rakeNow := false
	rakeTarget := cat.CurrentTarget
	rakeDot := cat.Rake.CurDot()

	for _, aoeTarget := range sim.Encounter.TargetUnits {
		rakeDot = cat.Rake.Dot(aoeTarget)
		canRakeTarget := !rakeDot.IsActive() || ((rakeDot.RemainingDuration(sim) < rakeDot.TickLength) && (!isClearcast || (rakeDot.RemainingDuration(sim) < time.Second)))

		if canRakeTarget {
			rakeNow = true
			rakeTarget = aoeTarget
			break
		}
	}

	if rakeNow && !roarNow {
		// Compare DPE versus Swipe to see if it's worth casting
		potentialRakeTicks := min(rakeDot.NumberOfTicks, int32(simTimeRemain/rakeDot.TickLength))
		expectedRakeDamage := cat.Rake.ExpectedInitialDamage(sim, rakeTarget) + cat.Rake.ExpectedTickDamage(sim, rakeTarget)*float64(potentialRakeTicks)
		rakeDPE := expectedRakeDamage / cat.Rake.DefaultCast.Cost
		expectedSwipeDamage, swipeDPE := cat.calcExpectedSwipeDamage(sim)

		if sim.Log != nil {
			cat.Log(sim, "Rake DPE = %.1f, Swipe DPE = %.1f", rakeDPE, swipeDPE)
		}

		rakeNow = core.Ternary(isClearcast, expectedRakeDamage > expectedSwipeDamage, rakeDPE > swipeDPE)
	}

	// Mangle check
	mangleNow := false
	mangleTarget := cat.CurrentTarget
	bleedAura := cat.bleedAura

	for _, aoeTarget := range sim.Encounter.TargetUnits {
		rakeDot = cat.Rake.Dot(aoeTarget)
		bleedAura = aoeTarget.GetExclusiveEffectCategory(core.BleedEffectCategory).GetActiveAura()
		canMangleTarget := rakeDot.IsActive() && !bleedAura.IsActive()

		if canMangleTarget {
			mangleNow = true
			mangleTarget = aoeTarget
			break
		}
	}

	if mangleNow && !roarNow && !rakeNow {
		// Compare Swipe damage to 30% of the max Rake ticks possible on this target before it dies
		currentRakeTicksRemaining := min(rakeDot.NumTicksRemaining(sim), int32(simTimeRemain/rakeDot.TickLength))
		newRakesPossible := max(0, int32((simTimeRemain-rakeDot.RemainingDuration(sim))/rakeDot.Duration))
		mangleRakeContribution := 0.3 * cat.Rake.ExpectedTickDamage(sim, mangleTarget) * float64(currentRakeTicksRemaining+newRakesPossible*(rakeDot.NumberOfTicks+1))
		rawMangleDamage := cat.MangleCat.ExpectedInitialDamage(sim, mangleTarget)
		expectedMangleDamage := rawMangleDamage + mangleRakeContribution
		mangleDPE := expectedMangleDamage / cat.MangleCat.DefaultCast.Cost
		expectedSwipeDamage, swipeDPE := cat.calcExpectedSwipeDamage(sim)

		if sim.Log != nil {
			cat.Log(sim, "Effective Mangle DPE = %.1f, Swipe DPE = %.1f", mangleDPE, swipeDPE)
		}

		mangleNow = core.Ternary(isClearcast, expectedMangleDamage >= expectedSwipeDamage, mangleDPE >= swipeDPE)
	}

	timeToNextAction := time.Duration(0)

	if roarNow {
		if cat.SavageRoar.CanCast(sim, cat.CurrentTarget) {
			cat.SavageRoar.Cast(sim, nil)
			return false, 0
		}
		timeToNextAction = core.DurationFromSeconds((cat.CurrentSavageRoarCost() - curEnergy) / regenRate)
	} else if rakeNow {
		if cat.Rake.CanCast(sim, rakeTarget) {
			cat.Rake.Cast(sim, rakeTarget)
			return false, 0
		}
		timeToNextAction = core.DurationFromSeconds((cat.CurrentRakeCost() - curEnergy) / regenRate)
	} else if mangleNow {
		if cat.MangleCat.CanCast(sim, mangleTarget) {
			cat.MangleCat.Cast(sim, mangleTarget)
			return false, 0
		}
		timeToNextAction = core.DurationFromSeconds((cat.CurrentMangleCatCost() - curEnergy) / regenRate)
	} else {
		if cat.SwipeCat.CanCast(sim, cat.CurrentTarget) {
			cat.SwipeCat.Cast(sim, cat.CurrentTarget)
			return false, 0
		}
		timeToNextAction = core.DurationFromSeconds((cat.CurrentSwipeCatCost() - curEnergy) / regenRate)
	}

	// Schedule next action based on any upcoming timers
	nextAction := sim.CurrentTime + timeToNextAction

	roarRefreshPending := cat.SavageRoarAura.IsActive() && (cat.SavageRoarAura.RemainingDuration(sim) < simTimeRemain-cat.ReactionTime) && (curCp >= 1)
	if roarRefreshPending {
		nextAction = min(nextAction, cat.SavageRoarAura.ExpiresAt())
	}

	for _, aoeTarget := range sim.Encounter.TargetUnits {
		rakeDot = cat.Rake.Dot(aoeTarget)
		rakeRefreshPending := rakeDot.IsActive() && (rakeDot.RemainingDuration(sim) < simTimeRemain-rakeDot.TickLength)

		if rakeRefreshPending && (rakeDot.RemainingDuration(sim) > rakeDot.TickLength) {
			nextAction = min(nextAction, rakeDot.ExpiresAt()-rakeDot.TickLength)
			bleedAura = aoeTarget.GetExclusiveEffectCategory(core.BleedEffectCategory).GetActiveAura()

			if bleedAura.IsActive() && (bleedAura.RemainingDuration(sim) < simTimeRemain-time.Second) {
				nextAction = min(nextAction, bleedAura.ExpiresAt())
			}
		}
	}

	return true, nextAction
}
