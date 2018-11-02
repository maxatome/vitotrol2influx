package main

import (
	"github.com/maxatome/go-vitotrol"
)

// ComputedAttr describes a custom attribute that is computed using
// several other attributes.
type ComputedAttr struct {
	// Attrs lists the attributes needed in Compute function.
	Attrs []vitotrol.AttrID
	// Compute is the function used to compute the value of this custom
	// attribute.
	Compute func(vdev *vitotrol.Device) interface{}
}

var computedAttrs = map[string]ComputedAttr{
	"ComputedSetpointTemp": {
		Attrs: []vitotrol.AttrID{
			vitotrol.EnergySavingMode,
			vitotrol.HeatReducedTemp,
			vitotrol.PartyMode,
			vitotrol.PartyModeTemp,
			vitotrol.OperatingModeCurrent,
			vitotrol.HeatNormalTemp,
		},
		Compute: func(vdev *vitotrol.Device) interface{} {
			if getAttrValue(vdev, vitotrol.EnergySavingMode).(int) != 0 {
				return getAttrValue(vdev, vitotrol.HeatReducedTemp)
			}
			if getAttrValue(vdev, vitotrol.PartyMode).(int) != 0 {
				return getAttrValue(vdev, vitotrol.PartyModeTemp)
			}
			switch getAttrValue(vdev, vitotrol.OperatingModeCurrent).(int) {
			case 0: // stand-by
				return float64(0) // probably around 4°C, not 0 in reality...
			case 1: // reduced
				return getAttrValue(vdev, vitotrol.HeatReducedTemp)
			default: // normal + continuous normal
				// reduced seems to not include enabled EnergySavingMode
				return getAttrValue(vdev, vitotrol.HeatNormalTemp)
			}
		},
	},
}
