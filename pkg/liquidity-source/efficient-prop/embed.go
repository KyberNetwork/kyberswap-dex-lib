package efficientprop

import _ "embed"

// The listing door's ABI, trimmed to quote(), quoteFillable(), its immutables and Swapped.
//
//go:embed abis/ListingDoor.json
var doorJSON []byte

// The front's ABI, trimmed to doorView(uint24), clock(), deskCellOf(uint32), CELL_CODEHASH(), the tip-taking
// quoteFrom(address,address,uint256,uint24,uint256,address) and the events naming a listing or a desk, its desks'
// openings, its pause and unpause of every fill, a pair's retirement and an asset module's binding.
//
//go:embed abis/Front.json
var frontJSON []byte

// The listing-door factory's ABI, trimmed to FRONT(), DOOR_CODEHASH() and doorOf().
//
//go:embed abis/ListingDoors.json
var factoryJSON []byte

// A desk's quote contract's logs: DeskPush() and LaneFill(), with no data, and on a builder build the fill's own
// Swapped (IPropAMM's event, which the listing door logs on a storage build).
//
//go:embed abis/DeskCell.json
var cellJSON []byte
