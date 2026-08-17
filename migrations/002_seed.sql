INSERT INTO checklist_templates(template_id, version, published, payload)
VALUES ('standard', 1, true, '{"template_id":"standard","version":1,"name":"生产发布清单","published":true,"items":[{"id":"backup","title":"备份已确认","required":true},{"id":"monitoring","title":"监控已接入","required":true}]}')
ON CONFLICT (template_id, version) DO NOTHING;
