package druid

import (
	"time"

	"github.com/wowsims/cata/sim/core"
	"github.com/wowsims/cata/sim/core/proto"
)

func (druid *Druid) registerMangleBearSpell() {
	mangleAuras := druid.NewEnemyAuraArray(core.MangleAura)
	glyphBonus := core.TernaryFloat64(druid.HasPrimeGlyph(proto.DruidPrimeGlyph_GlyphOfMangle), 1.1, 1.0)

	druid.MangleBear = druid.RegisterSpell(Bear, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 48564},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagIncludeTargetBonusDamage | core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost:   15,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: time.Duration(float64(time.Second) * 6),
			},
		},

		DamageMultiplier: 1.9 * glyphBonus,
		CritMultiplier:   druid.DefaultMeleeCritMultiplier(),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 3306.0/1.9 +
				spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower())

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				mangleAuras.Get(target).Activate(sim)
			} else {
				spell.IssueRefund(sim)
			}

			if druid.BerserkAura.IsActive() {
				spell.CD.Reset()
			}

			// Preferentially consume Berserk procs over Clearcasting procs
			if druid.BerserkProcAura.IsActive() {
				druid.BerserkProcAura.Deactivate(sim)
			} else if druid.ClearcastingAura.IsActive() {
				druid.ClearcastingAura.Deactivate(sim)
			}
		},

		RelatedAuras: []core.AuraArray{mangleAuras},
	})
}

func (druid *Druid) registerMangleCatSpell() {
	mangleAuras := druid.NewEnemyAuraArray(core.MangleAura)
	glyphBonus := core.TernaryFloat64(druid.HasPrimeGlyph(proto.DruidPrimeGlyph_GlyphOfMangle), 1.1, 1.0)
	hasBloodletting := druid.HasPrimeGlyph(proto.DruidPrimeGlyph_GlyphOfBloodletting)

	druid.MangleCat = druid.RegisterSpell(Cat, core.SpellConfig{
		ActionID:    core.ActionID{SpellID: 33876},
		SpellSchool: core.SpellSchoolPhysical,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagIncludeTargetBonusDamage | core.SpellFlagAPL,

		EnergyCost: core.EnergyCostOptions{
			Cost:   35.0,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: time.Second,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 5.4 * glyphBonus,
		CritMultiplier:   druid.DefaultMeleeCritMultiplier(),
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			baseDamage := 302.0/5.4 +
				spell.Unit.MHWeaponDamage(sim, spell.MeleeAttackPower())

			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeWeaponSpecialHitAndCrit)

			if result.Landed() {
				druid.AddComboPoints(sim, 1, spell.ComboPointMetrics())
				mangleAuras.Get(target).Activate(sim)

				// Mangle (Cat) can also extend Rip in Cata
				if hasBloodletting {
					druid.ApplyBloodletting(target)
				}
			} else {
				spell.IssueRefund(sim)
			}
		},

		ExpectedInitialDamage: func(sim *core.Simulation, target *core.Unit, spell *core.Spell, _ bool) *core.SpellResult {
			baseDamage := 302.0/5.4 + spell.Unit.AutoAttacks.MH().CalculateAverageWeaponDamage(spell.MeleeAttackPower())
			return spell.CalcDamage(sim, target, baseDamage, spell.OutcomeExpectedMeleeWeaponSpecialHitAndCrit)
		},

		RelatedAuras: []core.AuraArray{mangleAuras},
	})
}

func (druid *Druid) CurrentMangleCatCost() float64 {
	return druid.MangleCat.ApplyCostModifiers(druid.MangleCat.DefaultCast.Cost)
}

func (druid *Druid) IsMangle(spell *core.Spell) bool {
	if druid.MangleBear != nil && druid.MangleBear.IsEqual(spell) {
		return true
	} else if druid.MangleCat != nil && druid.MangleCat.IsEqual(spell) {
		return true
	}
	return false
}
