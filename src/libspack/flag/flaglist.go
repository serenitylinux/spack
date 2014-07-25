package flag

import (

)
//TODO map[string]Flag
type FlagList []Flag

func (l FlagList) String() string {
	str := ""
	for i, flag := range l {
		str += flag.String()
		if i != len(l)-1 {
			str += " "
		}
	}
	return str
}
func (l *FlagList) IsSubSet(ol FlagList) bool {
	for _, flag := range *l {
		found := false
		for _, oflag := range ol {
			if oflag.Name == flag.Name {
				found = oflag.Enabled == flag.Enabled
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (l *FlagList) Contains(f string) (*Flag, bool) {
	for _, flag := range *l {
		if flag.Name == f {
			return &flag, true
		}
	}
	return nil, false
}

func (l *FlagList) IsEnabled(f string) bool {
	for _, flag := range *l {
		if flag.Name == f {
			return flag.Enabled
		}
	}
	return false
}

func (l *FlagList) Append(f Flag) {
	*l = append(*l, f)
}

func (l *FlagList) Clone() (*FlagList) {
	newl := make(FlagList, len(*l))
	
	for i, flag := range *l {
		newl[i] = Flag { flag.Name, flag.Enabled }
	}
	
	return &newl
}