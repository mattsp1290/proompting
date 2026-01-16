#!/usr/bin/env python3
"""
migrate-tasks-to-beads.py - Preview/fast-path migration (optional)

This script attempts to parse common YAML structures and generate bd commands.
For reliable migration of ANY YAML structure, use migrate-tasks-to-beads.sh instead,
which uses Claude Code to interpret the YAML.

Usage:
    # Preview what would be migrated (dry run)
    python migrate-tasks-to-beads.py tasks.yaml

    # Generate migration script
    python migrate-tasks-to-beads.py tasks.yaml -o migrate.sh

    # For reliable migration of any structure, use Claude:
    ./migrate-tasks-to-beads.sh tasks.yaml --verify
"""

import argparse
import json
import re
import shlex
import sys
from pathlib import Path

try:
    import yaml
    YAML_AVAILABLE = True
except ImportError:
    YAML_AVAILABLE = False


def slugify(text: str) -> str:
    """Convert text to a URL-safe slug."""
    text = text.lower()
    text = re.sub(r'[^a-z0-9]+', '-', text)
    return text.strip('-')


def priority_to_beads(priority: str) -> int:
    """Convert priority string to beads priority number."""
    if not isinstance(priority, str):
        return 2
    mapping = {'critical': 0, 'high': 1, 'medium': 2, 'low': 3}
    return mapping.get(priority.lower(), 2)


def status_to_beads(status: str) -> str:
    """Convert status string to beads status."""
    if not isinstance(status, str):
        return 'open'
    mapping = {
        'pending': 'open', 'in-progress': 'in_progress',
        'in_progress': 'in_progress', 'completed': 'closed', 'done': 'closed'
    }
    return mapping.get(status.lower(), 'open')


def escape_shell(text: str) -> str:
    """Escape text for shell commands."""
    quoted = shlex.quote(text)
    if quoted.startswith("'") and quoted.endswith("'"):
        return quoted[1:-1]
    return quoted


def extract_tasks(data: dict) -> list:
    """
    Attempt to extract tasks from various YAML structures.
    Returns list of task dicts with: id, name, priority, status, dependencies, label
    """
    tasks = []

    # Try common structures
    # 1. phases[].tasks[]
    for phase in data.get('phases', []):
        if isinstance(phase, dict) and 'tasks' in phase:
            phase_name = phase.get('name', phase.get('id', 'default'))
            label = slugify(str(phase_name).replace('Phase ', '').split(':')[0])
            for task in phase.get('tasks', []):
                if isinstance(task, dict):
                    tasks.append({
                        'id': task.get('id', f'task-{len(tasks)}'),
                        'name': task.get('name', task.get('description', 'Unnamed task'))[:100],
                        'priority': priority_to_beads(task.get('priority', 'medium')),
                        'status': status_to_beads(task.get('status', 'pending')),
                        'dependencies': task.get('dependencies', []),
                        'label': label
                    })

    # 2. Top-level tasks[]
    if not tasks:
        for task in data.get('tasks', []):
            if isinstance(task, dict):
                task_id = task.get('id', f'task-{len(tasks)}')
                label = task_id.split('-')[0] if '-' in task_id else 'default'
                tasks.append({
                    'id': task_id,
                    'name': task.get('name', task.get('description', 'Unnamed task'))[:100],
                    'priority': priority_to_beads(task.get('priority', 'medium')),
                    'status': status_to_beads(task.get('status', 'pending')),
                    'dependencies': task.get('dependencies', []),
                    'label': label
                })

    # 3. phase1_xxx, phase2_xxx keys
    if not tasks:
        phase_keys = sorted([k for k in data.keys() if k.startswith('phase') and len(k) > 5 and k[5].isdigit()])
        for phase_key in phase_keys:
            phase_data = data.get(phase_key, {})
            if not isinstance(phase_data, dict):
                continue
            label = slugify(phase_data.get('name', phase_key))
            phase_tasks = phase_data.get('tasks', {})
            if not phase_tasks:
                meta = {'name', 'priority', 'estimated_hours', 'description', 'dependencies', 'purpose'}
                phase_tasks = {k: v for k, v in phase_data.items() if isinstance(v, dict) and k not in meta}
            for task_key, task_data in phase_tasks.items():
                if isinstance(task_data, dict):
                    name = task_data.get('name', task_data.get('description', task_key))
                    if isinstance(name, str) and len(name) > 100:
                        name = name[:100] + '...'
                    tasks.append({
                        'id': task_key,
                        'name': name,
                        'priority': priority_to_beads(task_data.get('priority', 'medium')),
                        'status': status_to_beads(task_data.get('status', 'pending')),
                        'dependencies': task_data.get('dependencies', []),
                        'label': label
                    })

    return tasks


def generate_script(tasks: list, project_name: str) -> str:
    """Generate shell script to create beads."""
    lines = [
        '#!/bin/bash',
        f'# Beads migration - {project_name}',
        '# Generated by migrate-tasks-to-beads.py',
        '# NOTE: For complex YAML, use migrate-tasks-to-beads.sh with Claude',
        '',
        'set -e',
        '',
        'if ! command -v bd &> /dev/null; then',
        '    echo "Error: bd CLI not found"',
        '    exit 1',
        'fi',
        '',
        '[ ! -d ".beads" ] && bd init',
        '',
        f'echo "Creating {len(tasks)} beads..."',
        ''
    ]

    id_map = {}
    for task in tasks:
        var = f'BEAD_{task["id"].upper().replace("-", "_").replace(":", "_")}'
        id_map[task['id']] = var
        title = escape_shell(str(task['name']))
        lines.append(f'bd create "{title}" -p {task["priority"]} --label {task["label"]}')
        lines.append(f'{var}=$(bd list --json | jq -r ".[-1].id")')
        lines.append('')

    # Dependencies
    lines.append('echo "Adding dependencies..."')
    for task in tasks:
        for dep in task.get('dependencies', []):
            if dep in id_map:
                lines.append(f'bd dep add ${id_map[task["id"]]} ${id_map[dep]} 2>/dev/null || true')

    # Mark completed
    completed = [t for t in tasks if t['status'] == 'closed']
    if completed:
        lines.append('')
        lines.append('echo "Marking completed..."')
        for task in completed:
            lines.append(f'bd update ${id_map[task["id"]]} --status closed 2>/dev/null || true')

    lines.extend(['', f'echo "Done: {len(tasks)} beads created"'])
    return '\n'.join(lines)


def main():
    parser = argparse.ArgumentParser(description='Preview/generate beads migration (use .sh for reliable migration)')
    parser.add_argument('yaml_file', type=Path, help='Path to tasks YAML')
    parser.add_argument('-o', '--output', type=Path, help='Write script to file')
    parser.add_argument('--json', action='store_true', help='Output JSON instead of shell script')
    args = parser.parse_args()

    if not args.yaml_file.exists():
        print(f"Error: {args.yaml_file} not found", file=sys.stderr)
        sys.exit(1)

    if not YAML_AVAILABLE:
        print("Error: PyYAML required. Install: pip install pyyaml", file=sys.stderr)
        print("Or use: ./migrate-tasks-to-beads.sh (Claude-based, no dependencies)", file=sys.stderr)
        sys.exit(1)

    with open(args.yaml_file) as f:
        data = yaml.safe_load(f)

    # Extract project name
    meta = data.get('metadata', data.get('project', {}))
    if isinstance(meta, dict):
        project = meta.get('project', meta.get('project_name', meta.get('name', 'unknown')))
    else:
        project = 'unknown'

    tasks = extract_tasks(data)

    if not tasks:
        print(f"Warning: No tasks found. YAML structure may not be supported.", file=sys.stderr)
        print(f"Use ./migrate-tasks-to-beads.sh for reliable migration.", file=sys.stderr)
        sys.exit(1)

    if args.json:
        output = json.dumps({'project': project, 'tasks': tasks}, indent=2)
    else:
        output = generate_script(tasks, project)

    if args.output:
        args.output.write_text(output)
        if not args.json:
            args.output.chmod(0o755)
        print(f"Written to {args.output} ({len(tasks)} tasks)", file=sys.stderr)
    else:
        print(output)


if __name__ == '__main__':
    main()
