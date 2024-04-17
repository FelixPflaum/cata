package mage

import (
	"time"

	"github.com/wowsims/cata/sim/core"
	"github.com/wowsims/cata/sim/core/proto"
)

func (mage *Mage) registerPyroblastSpell() {
	if mage.Spec != (proto.Spec_SpecFireMage) {
		return
	}

	hasT8_4pc := mage.HasSetBonus(ItemSetKirinTorGarb, 4)

	var pyroblastDot *core.Spell
	/* implement when debuffs updated
	var CMProcChance float64
	if mage.Talents.CriticalMass > 0 {
		CMProcChance = float64(mage.Talents.CriticalMass) / 3.0
		//TODO double check how this works
		mage.CriticalMassAuras = mage.NewEnemyAuraArray(core.CriticalMassAura)
		mage.CritDebuffCategories = mage.GetEnemyExclusiveCategories(core.SpellCritEffectCategory)
		mage.Pyroblast.RelatedAuras = append(mage.Pyroblast.RelatedAuras, mage.CriticalMassAuras)
	} */

	pyroConfig := core.SpellConfig{
		ActionID:     core.ActionID{SpellID: 11366},
		SpellSchool:  core.SpellSchoolFire,
		ProcMask:     core.ProcMaskSpellDamage,
		Flags:        SpellFlagMage | HotStreakSpells | core.SpellFlagAPL,
		MissileSpeed: 24,

		ManaCost: core.ManaCostOptions{
			BaseCost:   0.17,
			Multiplier: core.TernaryFloat64(mage.HotStreakAura.IsActive(), 0, 1),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * 3500,
			},
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				if mage.HotStreakAura.IsActive() {
					cast.CastTime = 0
					if !hasT8_4pc || sim.Proc(T84PcProcChance, "MageT84PC") {
						mage.HotStreakAura.Deactivate(sim)
					}
				}
			},
		},

		DamageMultiplier: 1,
		DamageMultiplierAdditive: 1 +
			.01*float64(mage.Talents.FirePower),
		CritMultiplier:   mage.DefaultSpellCritMultiplier(),
		BonusCoefficient: 1.545,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "PyroblastDoT",
			},
			NumberOfTicks: 4,
			TickLength:    time.Second * 3,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				dot.SnapshotBaseDamage = 0.175 * mage.ScalingBaseDamage
				dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(dot.Spell.Unit.AttackTables[target.UnitIndex])
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				pyroblastDot.SpellMetrics[target.UnitIndex].Hits++
			},
			BonusCoefficient: 0.180,
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 1.5 * mage.ScalingBaseDamage
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				if result.Landed() {
					pyroblastDot.Dot(target).Apply(sim)
					//pyroblastDot.SpellMetrics[target.UnitIndex].Hits++
					//pyroblastDot.SpellMetrics[target.UnitIndex].Casts = 0
					/* The 2 above metric changes should show how many ticks land
					without affecting the overall pyroblast cast metric
					*/
				}
				spell.DealDamage(sim, result)
			})
		},
	}
	// Unsure about the implementation of the below, but just trusting it since it existed here
	mage.Pyroblast = mage.RegisterSpell(pyroConfig)

	dotConfig := pyroConfig
	dotConfig.ActionID = dotConfig.ActionID.WithTag(1)
	pyroblastDot = mage.RegisterSpell(dotConfig)
}
