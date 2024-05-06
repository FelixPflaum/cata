package core

// **************************************
// AUTO GENERATED BY BASE_STATS_PARSER.PY
// **************************************

import (
	"github.com/wowsims/cata/sim/core/proto"
	"github.com/wowsims/cata/sim/core/stats"
)

const ExpertisePerQuarterPercentReduction = 30.027197
const HasteRatingPerHastePercent = 128.057160
const CritRatingPerCritChance = 179.280040
const MeleeHitRatingPerHitChance = 120.108800
const SpellHitRatingPerHitChance = 102.445740
const DefenseRatingPerDefense = 19.208574
const DodgeRatingPerDodgeChance = 176.718900
const ParryRatingPerParryChance = 176.718900
const BlockRatingPerBlockChance = 88.359444
const ResilienceRatingPerCritReductionChance = 0.000000
const MasteryRatingPerMasteryPoint = 179.280040

var CritPerAgiMaxLevel = map[proto.Class]float64{
	proto.Class_ClassUnknown:     0.0,
	proto.Class_ClassWarrior:     0.0041,
	proto.Class_ClassPaladin:     0.0049,
	proto.Class_ClassHunter:      0.0031,
	proto.Class_ClassRogue:       0.0031,
	proto.Class_ClassPriest:      0.0049,
	proto.Class_ClassDeathKnight: 0.0041,
	proto.Class_ClassShaman:      0.0031,
	proto.Class_ClassMage:        0.0050,
	proto.Class_ClassWarlock:     0.0051,
	proto.Class_ClassDruid:       0.0031,
}
var CritPerIntMaxLevel = map[proto.Class]float64{
	proto.Class_ClassUnknown:     0.0,
	proto.Class_ClassWarrior:     0.0000,
	proto.Class_ClassPaladin:     0.0015,
	proto.Class_ClassHunter:      0.0000,
	proto.Class_ClassRogue:       0.0000,
	proto.Class_ClassPriest:      0.0015,
	proto.Class_ClassDeathKnight: 0.0000,
	proto.Class_ClassShaman:      0.0015,
	proto.Class_ClassMage:        0.0015,
	proto.Class_ClassWarlock:     0.0015,
	proto.Class_ClassDruid:       0.0015,
}
var ExtraClassBaseStats = map[proto.Class]stats.Stats{
	proto.Class_ClassUnknown: {},
	proto.Class_ClassWarrior: {
		stats.Mana:      0.0000,
		stats.SpellCrit: 0.0000 * CritRatingPerCritChance,
		stats.MeleeCrit: 0.0000 * CritRatingPerCritChance,
	},
	proto.Class_ClassPaladin: {
		stats.Mana:      23422.0000,
		stats.SpellCrit: 3.3355 * CritRatingPerCritChance,
		stats.MeleeCrit: 0.6520 * CritRatingPerCritChance,
	},
	proto.Class_ClassHunter: {
		stats.Mana:      0.0000,
		stats.SpellCrit: 0.0000 * CritRatingPerCritChance,
		stats.MeleeCrit: -1.5320 * CritRatingPerCritChance,
	},
	proto.Class_ClassRogue: {
		stats.Mana:      0.0000,
		stats.SpellCrit: 0.0000 * CritRatingPerCritChance,
		stats.MeleeCrit: -0.2950 * CritRatingPerCritChance,
	},
	proto.Class_ClassPriest: {
		stats.Mana:      20590.0000,
		stats.SpellCrit: 1.2375 * CritRatingPerCritChance,
		stats.MeleeCrit: 3.1765 * CritRatingPerCritChance,
	},
	proto.Class_ClassDeathKnight: {
		stats.Mana:      0.0000,
		stats.SpellCrit: 0.0000 * CritRatingPerCritChance,
		stats.MeleeCrit: 0.0000 * CritRatingPerCritChance,
	},
	proto.Class_ClassShaman: {
		stats.Mana:      23430.0000,
		stats.SpellCrit: 2.2010 * CritRatingPerCritChance,
		stats.MeleeCrit: 2.9220 * CritRatingPerCritChance,
	},
	proto.Class_ClassMage: {
		stats.Mana:      17418.0000,
		stats.SpellCrit: 0.9075 * CritRatingPerCritChance,
		stats.MeleeCrit: 3.4540 * CritRatingPerCritChance,
	},
	proto.Class_ClassWarlock: {
		stats.Mana:      20553.0000,
		stats.SpellCrit: 1.7000 * CritRatingPerCritChance,
		stats.MeleeCrit: 2.6220 * CritRatingPerCritChance,
	},
	proto.Class_ClassDruid: {
		stats.Mana:      18635.0000,
		stats.SpellCrit: 1.8515 * CritRatingPerCritChance,
		stats.MeleeCrit: 7.4755 * CritRatingPerCritChance,
	},
}
