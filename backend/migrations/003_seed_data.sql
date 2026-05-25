-- Seed Log Sources
-- ============================================================
-- PUSINGBERAT SIEM - Comprehensive Seed Data
-- Populates Sources, Rules, Events, and Alerts for Frontend Dev
-- ============================================================

-- 1. Seed Log Sources
INSERT INTO log_sources (id, name, file_path, log_type, status, description) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'System Auth Log', '/host/var/log/auth.log', 'syslog', 'active', 'Monitor system authentication events'),
('550e8400-e29b-41d4-a716-446655440002', 'Nginx Access Log', '/host/var/log/nginx/access.log', 'nginx', 'active', 'Monitor web traffic and HTTP errors'),
('550e8400-e29b-41d4-a716-446655440003', 'App Server Log', '/host/var/log/syslog', 'generic', 'active', 'Generic system logs')
ON CONFLICT (file_path) DO NOTHING;

-- 2. Seed Detection Rules
INSERT INTO rules (id, name, description, yaml_content, severity, enabled) VALUES
('6ba7b810-9dad-11d1-80b4-00c04fd430c1', 'SSH Brute Force', 'Detects repeated SSH failures', 'id: ssh-bf\nname: SSH Brute Force\nseverity: high\nlog_types: [syslog]\nconditions:\n  - field: message\n    operator: contains\n    value: "Failed password"', 'high', true),
('6ba7b810-9dad-11d1-80b4-00c04fd430c2', 'Sudo Auth Failure', 'Detects sudo failures', 'id: sudo-err\nname: Sudo Auth Failure\nseverity: medium\nlog_types: [syslog]\nconditions:\n  - field: message\n    operator: contains\n    value: "auth failure"', 'medium', true),
('6ba7b810-9dad-11d1-80b4-00c04fd430c3', 'Nginx 5xx Critical', 'Detects server side errors', 'id: nginx-5xx\nname: Nginx 5xx Critical\nseverity: critical\nlog_types: [nginx]\nconditions:\n  - field: status\n    operator: starts_with\n    value: "5"', 'critical', true),
('6ba7b810-9dad-11d1-80b4-00c04fd430c4', 'Web Scanner Detected', 'Detects common scan patterns', 'id: scan-01\nname: Web Scanner Detected\nseverity: low\nlog_types: [nginx]\nconditions:\n  - field: path\n    operator: contains\n    value: ".env"', 'low', true)
ON CONFLICT (name) DO NOTHING;

-- 3. Seed Events (Generate activity for the last 24 hours to populate charts)
INSERT INTO events (log_source_id, raw_line, message, hostname, process, log_level, event_time, received_at)
SELECT 
    CASE 
        WHEN (n % 3) = 0 THEN '550e8400-e29b-41d4-a716-446655440001'::UUID
        WHEN (n % 3) = 1 THEN '550e8400-e29b-41d4-a716-446655440002'::UUID
        ELSE '550e8400-e29b-41d4-a716-446655440003'::UUID
    END as log_source_id,
    'Sample log line content for event number ' || n as raw_line,
    'Event message ' || n as message,
    'pusingberat-host-01' as hostname,
    CASE WHEN (n % 2) = 0 THEN 'sshd' ELSE 'nginx' END as process,
    CASE WHEN (n % 10) = 0 THEN 'ERROR' ELSE 'INFO' END as log_level,
    NOW() - (n || ' minutes')::interval as event_time,
    NOW() - (n || ' minutes')::interval as received_at
FROM generate_series(1, 100) n;

-- 4. Seed Alerts (Populates the dashboard and severity charts)
-- Critical Alert (Unacknowledged)
INSERT INTO alerts (rule_id, rule_name, log_source_id, severity, title, description, raw_line, triggered_at)
VALUES (
    '6ba7b810-9dad-11d1-80b4-00c04fd430c3', 
    'Nginx 5xx Critical', 
    '550e8400-e29b-41d4-a716-446655440002', 
    'critical', 
    'Critical: Web Server Internal Error', 
    'Multiple 500 status codes detected on Nginx Access Log.', 
    '127.0.0.1 - - [23/May/2026:08:00:00 +0000] "GET /api/v1/data HTTP/1.1" 500 123',
    NOW() - INTERVAL '15 minutes'
);

-- High Alert (Unacknowledged)
INSERT INTO alerts (rule_id, rule_name, log_source_id, severity, title, description, raw_line, triggered_at)
VALUES (
    '6ba7b810-9dad-11d1-80b4-00c04fd430c1', 
    'SSH Brute Force', 
    '550e8400-e29b-41d4-a716-446655440001', 
    'high', 
    'High: SSH Brute Force Detected', 
    'Detected 5 failed login attempts within 60 seconds from 192.168.1.50', 
    'May 23 07:56:40 pusingberat-host-01 sshd[123]: Failed password for root',
    NOW() - INTERVAL '45 minutes'
);

-- Medium Alert (Acknowledged)
INSERT INTO alerts (rule_id, rule_name, log_source_id, severity, title, description, triggered_at, acknowledged, acknowledged_at)
VALUES (
    '6ba7b810-9dad-11d1-80b4-00c04fd430c2', 
    'Sudo Auth Failure', 
    '550e8400-e29b-41d4-a716-446655440001', 
    'medium', 
    'Medium: Unauthorized Sudo Attempt', 
    'User "guest" tried to execute /usr/bin/apt update via sudo.', 
    NOW() - INTERVAL '2 hours',
    true,
    NOW() - INTERVAL '1 hour'
);

-- Low Alerts (To fill the chart)
INSERT INTO alerts (rule_id, rule_name, log_source_id, severity, title, description, triggered_at)
VALUES 
(
    '6ba7b810-9dad-11d1-80b4-00c04fd430c4', 
    'Web Scanner Detected', 
    '550e8400-e29b-41d4-a716-446655440002', 
    'low', 
    'Low: Potential Reconnaissance', 
    'Request to common sensitive file path detected.', 
    NOW() - INTERVAL '4 hours'
),
(
    '6ba7b810-9dad-11d1-80b4-00c04fd430c4', 
    'Web Scanner Detected', 
    '550e8400-e29b-41d4-a716-446655440002', 
    'low', 
    'Low: Potential Reconnaissance', 
    'Request to common sensitive file path detected.', 
    NOW() - INTERVAL '5 hours'
);

-- Info Alert
INSERT INTO alerts (rule_id, rule_name, log_source_id, severity, title, description, triggered_at)
VALUES (
    NULL, 
    'Manual System Check', 
    '550e8400-e29b-41d4-a716-446655440003', 
    'info', 
    'Info: System Maintenance Scheduled', 
    'Automated seed notification for maintenance.', 
    NOW() - INTERVAL '12 hours'
);