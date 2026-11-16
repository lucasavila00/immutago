package rejectedconsumer

import "example.com/consumer/api"

func scalarPointers(levels *api.Levels, reply api.LevelReply) {
	levels.Current.Raise()
	levels.Values[0].Raise()
	reply.Current.Raise()
	bound := levels.Current.Raise
	bound()
	go reply.Current.Urgent()
	defer reply.Values[0].Urgent()
}
