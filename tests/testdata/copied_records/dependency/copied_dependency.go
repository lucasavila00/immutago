package dependency

import "example.com/consumer/badapi"

var value badapi.Alias

func Read() badapi.Alias { return value }
