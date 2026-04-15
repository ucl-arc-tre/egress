CREATE UNIQUE INDEX IF NOT EXISTS idx_file_approvals_unique_approval ON file_approvals(project_id, file_id, user_id);
