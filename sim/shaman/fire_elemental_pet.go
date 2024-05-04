package shaman

import (
	"math"
	"time"

	"github.com/wowsims/cata/sim/core"
	"github.com/wowsims/cata/sim/core/proto"
	"github.com/wowsims/cata/sim/core/stats"
)

type FireElemental struct {
	core.Pet

	FireBlast *core.Spell
	FireNova  *core.Spell

	maxFireBlastCasts int32
	maxFireNovaCasts  int32

	FireShieldAura *core.Aura

	shamanOwner *Shaman
}

func (shaman *Shaman) NewFireElemental(bonusSpellPower float64) *FireElemental {
	fireElemental := &FireElemental{
		Pet:               core.NewPet("Greater Fire Elemental", &shaman.Character, fireElementalPetBaseStats, shaman.fireElementalStatInheritance(), false, true),
		shamanOwner:       shaman,
		maxFireBlastCasts: 15,
		maxFireNovaCasts:  15,
	}
	fireElemental.EnableManaBar()
	fireElemental.EnableAutoAttacks(fireElemental, core.AutoAttackOptions{
		MainHand: core.Weapon{
			BaseDamageMin:  429, //Estimated from beta testing
			BaseDamageMax:  463, //Estimated from beta testing
			SwingSpeed:     2,
			CritMultiplier: 2.66, //Estimated from beta testing
			SpellSchool:    core.SpellSchoolFire,
		},
		AutoSwingMelee: true,
	})

	if bonusSpellPower > 0 {
		fireElemental.AddStat(stats.SpellPower, float64(bonusSpellPower)*0.5883)
		fireElemental.AddStat(stats.AttackPower, float64(bonusSpellPower)*0.7)
	}

	if shaman.Race == proto.Race_RaceDraenei {
		fireElemental.AddStats(stats.Stats{
			stats.MeleeHit:  -core.MeleeHitRatingPerHitChance,
			stats.SpellHit:  -core.SpellHitRatingPerHitChance,
			stats.Expertise: math.Floor(-core.SpellHitRatingPerHitChance * 0.79),
		})
	}

	fireElemental.OnPetEnable = fireElemental.enable
	fireElemental.OnPetDisable = fireElemental.disable

	shaman.AddPet(fireElemental)

	return fireElemental
}

func (fireElemental *FireElemental) enable(sim *core.Simulation) {
	fireElemental.FireShieldAura.Activate(sim)
}

func (fireElemental *FireElemental) disable(sim *core.Simulation) {
	fireElemental.FireShieldAura.Deactivate(sim)
}

func (fireElemental *FireElemental) GetPet() *core.Pet {
	return &fireElemental.Pet
}

func (fireElemental *FireElemental) Initialize() {

	fireElemental.registerFireBlast()
	fireElemental.registerFireNova()
	fireElemental.registerFireShieldAura()
}

func (fireElemental *FireElemental) Reset(_ *core.Simulation) {

}

func (fireElemental *FireElemental) ExecuteCustomRotation(sim *core.Simulation) {
	/*
		TODO this is a little dirty, can probably clean this up, the rotation might go through some more overhauls,
		the random AI is hard to emulate.
	*/
	target := fireElemental.CurrentTarget
	fireBlastCasts := fireElemental.FireBlast.SpellMetrics[0].Casts
	fireNovaCasts := fireElemental.FireNova.SpellMetrics[0].Casts

	if fireBlastCasts == fireElemental.maxFireBlastCasts && fireNovaCasts == fireElemental.maxFireNovaCasts {
		return
	}

	if fireElemental.FireNova.DefaultCast.Cost > fireElemental.CurrentMana() {
		return
	}

	random := sim.RandomFloat("Fire Elemental Pet Spell")

	//Melee the other 30%
	if random >= .65 {
		if !fireElemental.TryCast(sim, target, fireElemental.FireNova, fireElemental.maxFireNovaCasts) {
			fireElemental.TryCast(sim, target, fireElemental.FireBlast, fireElemental.maxFireBlastCasts)
		}
	} else if random >= .35 {
		if !fireElemental.TryCast(sim, target, fireElemental.FireBlast, fireElemental.maxFireBlastCasts) {
			fireElemental.TryCast(sim, target, fireElemental.FireNova, fireElemental.maxFireNovaCasts)
		}
	}

	if !fireElemental.GCD.IsReady(sim) {
		return
	}

	fireElemental.ExtendGCDUntil(sim, sim.CurrentTime+time.Second)
}

func (fireElemental *FireElemental) TryCast(sim *core.Simulation, target *core.Unit, spell *core.Spell, maxCastCount int32) bool {
	if maxCastCount == spell.SpellMetrics[0].Casts {
		return false
	}

	if !spell.Cast(sim, target) {
		return false
	}
	// all spell casts reset the elemental's swing timer
	fireElemental.AutoAttacks.StopMeleeUntil(sim, sim.CurrentTime+spell.CurCast.CastTime, false)
	return true
}

var fireElementalPetBaseStats = stats.Stats{
	stats.Mana:        8893, //Estimated from beta testing
	stats.Health:      4903, //Estimated from beta testing
	stats.Intellect:   0,
	stats.Stamina:     0,
	stats.SpellPower:  0, //Estimated
	stats.AttackPower: 0, //Estimated

	// TODO : Log digging and my own samples this seems to be around the 5% mark.
	stats.MeleeCrit: (5 + 1.8) * core.CritRatingPerCritChance,
	stats.SpellCrit: 2.61 * core.CritRatingPerCritChance,
}

func (shaman *Shaman) fireElementalStatInheritance() core.PetStatInheritance {
	return func(ownerStats stats.Stats) stats.Stats {
		ownerSpellHitChance := math.Floor(ownerStats[stats.SpellHit] / core.SpellHitRatingPerHitChance)
		spellHitRatingFromOwner := ownerSpellHitChance * core.SpellHitRatingPerHitChance

		return stats.Stats{
			stats.Stamina:     ownerStats[stats.Stamina] * 0.80,      //Estimated from beta testing
			stats.Intellect:   ownerStats[stats.Intellect] * 0.3198,  //Estimated from beta testing
			stats.SpellPower:  ownerStats[stats.SpellPower] * 0.5883, //Estimated from beta testing
			stats.AttackPower: ownerStats[stats.SpellPower] * 0.7,    //Estimated from beta testing

			stats.MeleeHit: spellHitRatingFromOwner,
			stats.SpellHit: spellHitRatingFromOwner,

			/*
				TODO working on figuring this out, getting close need more trials. will need to remove specific buffs,
				ie does not gain the benefit from draenei buff.
			*/
			stats.Expertise: math.Floor(spellHitRatingFromOwner * 27 / 16),
		}
	}
}
