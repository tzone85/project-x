CREATE INDEX IF NOT EXISTS idx_stories_req_id ON stories(req_id);
CREATE INDEX IF NOT EXISTS idx_stories_status ON stories(status);
CREATE INDEX IF NOT EXISTS idx_stories_req_status ON stories(req_id, status);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_escalations_story_id ON escalations(story_id);
