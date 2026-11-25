package everlongflamm

import "slices"

// defaultHookFamilies are the hook builds the protocol lists, routed when the config names no family: the swap
// hook builds the gas table measured (gasFamilies, generated with it), and the PropAMM front of each chain. Each hash
// is keccak256 of the build's runtime with its immutables (the ranges) zeroed, so every deployment of the build
// shares it whatever its constructor arguments; a config `hookFamilies` list replaces these. The swap hook states its
// output's shape, which settles its ladders. The fronts state none: this build routes no pair a front prices, listed
// or not, and keeps their entries for the PropAMM listings.
var defaultHookFamilies = slices.Concat(gasFamilies, frontFamilies)

var frontFamilies = []HookFamily{
	{
		Name:     "propamm-front-base",
		Front:    true,
		CodeHash: "0x441dcdec46a5acc18348319988b42adfe87d915702191280ac87209495bff964",
		Immutables: [][2]int{
			{114, 32}, {180, 32}, {1070, 32}, {1128, 32}, {1214, 32}, {1789, 32}, {1950, 32}, {2102, 32},
			{2291, 32}, {2417, 32}, {2475, 32}, {2533, 32}, {2623, 32}, {2986, 32}, {8311, 32}, {8630, 32},
			{8663, 32}, {8696, 32}, {8848, 32}, {9035, 32}, {14129, 32}, {14181, 32}, {19846, 32},
		},
	},
	{
		Name:     "propamm-front-ethereum",
		Front:    true,
		CodeHash: "0x4c6c3bdda414f39a82bf1f0109b5a2a55af6d6076ba18ea865abce9b5c3cf3e5",
		Immutables: [][2]int{
			{114, 32}, {180, 32}, {1217, 32}, {1275, 32}, {1361, 32}, {2033, 32}, {2194, 32}, {2346, 32},
			{2535, 32}, {2661, 32}, {2719, 32}, {2777, 32}, {2867, 32}, {3230, 32}, {8841, 32}, {9154, 32},
			{9187, 32}, {9220, 32}, {9372, 32}, {9517, 32}, {14554, 32}, {14606, 32}, {20411, 32},
		},
	},
	{
		Name:     "propamm-front-bnb",
		Front:    true,
		CodeHash: "0x28334baad461ee716922672eb385585730e51302cc9429e5970b51c1bc04615c",
		Immutables: [][2]int{
			{114, 32}, {180, 32}, {1217, 32}, {1275, 32}, {1361, 32}, {2033, 32}, {2194, 32}, {2346, 32},
			{2535, 32}, {2661, 32}, {2719, 32}, {2777, 32}, {2867, 32}, {3230, 32}, {8841, 32}, {9154, 32},
			{9187, 32}, {9220, 32}, {9372, 32}, {9517, 32}, {14554, 32}, {14606, 32}, {20411, 32},
		},
	},
}
