package flag

import (
	
)

type FlagSetList []FlagSet

func (fsl FlagSetList) String() string {
	res := ""
	for _, fs := range fsl {
		res += fs.String() + " "
	}
	return res
}

func (fsl *FlagSetList) Verify(list *FlagList) bool {
	for _, fs := range *fsl {
		if ! fs.Verify(list) {
			return false
		}
	}
	return true
}