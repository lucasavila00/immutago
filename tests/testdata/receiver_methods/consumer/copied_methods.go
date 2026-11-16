package consumer

import "example.com/consumer/api"

func urgent(levels *api.Levels, reply api.LevelReply) bool {
	_ = levels.Values[0].Urgent()
	_ = reply.Values[0].Urgent()
	_ = api.Level.Urgent(reply.Current)
	read := reply.Current.Urgent
	_ = read()
	return levels.Current.Urgent()
}
