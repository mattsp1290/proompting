#!/usr/bin/env python3
"""
migrate-tasks-to-beads.py

Converts a tasks.yaml file to Beads format, generating bd CLI commands
that create the equivalent task graph.

Usage:
    python migrate-tasks-to-beads.py tasks.yaml > setup-beads.sh
    python migrate-tasks-to-beads.py tasks.yaml --execute  # Run directly
    python migrate-tasks-to-beads.py tasks.yaml --dry-run  # Preview only
    python migrate-tasks-to-beads.py tasks.yaml --fallback-claude  # Use Claude on parse failure
"""

import argparse
import json
import re
import shlex
import subprocess
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
    """Convert priority string to beads priority number (0=critical, 3=low)."""
    mapping = {
        'critical': 0,
        'high': 1,
        'medium': 2,
        'low': 3
    }
    return mapping.get(priority.lower(), 2)


def status_to_beads(status: str) -> str:
    """Convert status string to beads status."""
    mapping = {
        'pending': 'open',
        'in-progress': 'in_progress',
        'in_progress': 'in_progress',
        'completed': 'closed',
        'done': 'closed'
    }
    return mapping.get(status.lower(), 'open')


def escape_shell(text: str) -> str:
    """Escape text for shell commands using shlex for robust handling."""
    # shlex.quote returns a shell-escaped version of the string
    # We strip the outer quotes since we'll add our own in the command
    quoted = shlex.quote(text)
    # If shlex added single quotes, extract the inner content
    if quoted.startswith("'") and quoted.endswith("'"):
        return quoted[1:-1]
    return quoted


def parse_tasks_yaml(yaml_path: Path) -> dict:
    """Parse the tasks.yaml file and return structured data.

    Supports three YAML structures:
    1. Tasks nested under phases[].tasks[] (e.g., sorcery-backend, birbgame)
    2. Tasks at top-level under tasks[] with separate phases[] for metadata (e.g., retro-graph)
    3. Phase keys like phase1_xxx:, phase2_xxx: with nested task objects (e.g., birb-home)
    """
    with open(yaml_path) as f:
        data = yaml.safe_load(f)

    tasks = []
    task_id_map = {}  # Map from yaml id to sequential number for variable names

    # Extract metadata - support both 'metadata' and 'project' keys
    metadata = data.get('metadata', {})
    if not metadata and 'project' in data and isinstance(data['project'], dict):
        metadata = data['project']
    project_name = metadata.get('project', metadata.get('project_name', metadata.get('name', 'unknown')))

    # Build phase name lookup for top-level tasks structure
    phase_names = {}
    phases = data.get('phases', [])
    for phase_idx, phase in enumerate(phases):
        phase_id = phase.get('id', f'phase-{phase_idx}')
        phase_name = phase.get('name', phase_id)
        phase_names[phase_id] = slugify(phase_name.replace('Phase ', '').split(':')[0])

    # Check for Structure 3: phase1_xxx, phase2_xxx keys
    phase_keys = [k for k in data.keys() if k.startswith('phase') and k[5:6].isdigit()]

    # Check if tasks are nested under phases or at top level
    has_nested_tasks = any(phase.get('tasks') for phase in phases)
    top_level_tasks = data.get('tasks', [])

    if phase_keys:
        # Structure 3: Phase keys like phase1_setup, phase2_basic_infra
        for phase_key in sorted(phase_keys):
            phase_data = data[phase_key]
            if not isinstance(phase_data, dict):
                continue

            phase_name = phase_data.get('name', phase_key)
            phase_label = slugify(phase_name.replace('Phase ', '').split(':')[0])

            # Tasks can be under 'tasks' dict or directly as keys
            phase_tasks = phase_data.get('tasks', {})
            if not phase_tasks:
                # Look for task-like keys (not standard metadata keys)
                meta_keys = {'name', 'priority', 'estimated_hours', 'description', 'dependencies'}
                phase_tasks = {k: v for k, v in phase_data.items()
                              if isinstance(v, dict) and k not in meta_keys}

            for task_key, task_data in phase_tasks.items():
                if not isinstance(task_data, dict):
                    continue

                task_id = task_key
                task_id_map[task_id] = f'BEAD_{task_id.upper().replace("-", "_").replace(":", "_")}'

                # Handle different field names
                task_name = task_data.get('name', task_data.get('description', task_id))
                if isinstance(task_name, str) and len(task_name) > 100:
                    task_name = task_name[:100] + '...'

                tasks.append({
                    'id': task_id,
                    'var_name': task_id_map[task_id],
                    'name': task_name,
                    'description': task_data.get('description', ''),
                    'priority': priority_to_beads(task_data.get('priority', 'medium')),
                    'status': status_to_beads(task_data.get('status', 'pending')),
                    'dependencies': task_data.get('dependencies', []),
                    'label': phase_label,
                    'references': task_data.get('references', [])
                })

    elif has_nested_tasks:
        # Structure 1: Tasks nested under phases[].tasks[]
        for phase_idx, phase in enumerate(phases):
            phase_name = phase.get('name', phase.get('id', f'phase-{phase_idx}'))
            phase_label = slugify(phase_name.replace('Phase ', '').split(':')[0])

            for task in phase.get('tasks', []):
                task_id = task.get('id', f'task-{len(tasks)}')
                task_id_map[task_id] = f'BEAD_{task_id.upper().replace("-", "_")}'

                tasks.append({
                    'id': task_id,
                    'var_name': task_id_map[task_id],
                    'name': task.get('name', task_id),
                    'description': task.get('description', ''),
                    'priority': priority_to_beads(task.get('priority', 'medium')),
                    'status': status_to_beads(task.get('status', 'pending')),
                    'dependencies': task.get('dependencies', []),
                    'label': phase_label,
                    'references': task.get('references', [])
                })
    elif top_level_tasks:
        # Structure 2: Tasks at top level under tasks[]
        # Try to infer phase from task id prefix or use a default
        for task in top_level_tasks:
            task_id = task.get('id', f'task-{len(tasks)}')
            task_id_map[task_id] = f'BEAD_{task_id.upper().replace("-", "_")}'

            # Try to determine phase label from task id or phase reference
            phase_ref = task.get('phase', '')
            if phase_ref and phase_ref in phase_names:
                phase_label = phase_names[phase_ref]
            else:
                # Infer from task id prefix (e.g., 'research-foo' -> 'research')
                parts = task_id.split('-')
                phase_label = parts[0] if parts else 'default'

            tasks.append({
                'id': task_id,
                'var_name': task_id_map[task_id],
                'name': task.get('name', task_id),
                'description': task.get('description', ''),
                'priority': priority_to_beads(task.get('priority', 'medium')),
                'status': status_to_beads(task.get('status', 'pending')),
                'dependencies': task.get('dependencies', []),
                'label': phase_label,
                'references': task.get('references', [])
            })

    return {
        'project': project_name,
        'metadata': metadata,
        'tasks': tasks,
        'task_id_map': task_id_map
    }


def generate_shell_script(data: dict) -> str:
    """Generate a shell script that creates all beads."""
    lines = [
        '#!/bin/bash',
        f'# Beads migration from tasks.yaml',
        f'# Project: {data["project"]}',
        f'# Generated by migrate-tasks-to-beads.py',
        '',
        'set -e',
        '',
        '# Check if bd is installed',
        'if ! command -v bd &> /dev/null; then',
        '    echo "Error: bd CLI not found. Install with: npm install -g @beads/bd"',
        '    exit 1',
        'fi',
        '',
        '# Initialize beads if needed',
        'if [ ! -d ".beads" ]; then',
        '    echo "Initializing Beads..."',
        '    bd init',
        'fi',
        '',
        'echo "Creating beads from tasks.yaml..."',
        ''
    ]

    # Group tasks by label (phase)
    by_label = {}
    for task in data['tasks']:
        label = task['label']
        if label not in by_label:
            by_label[label] = []
        by_label[label].append(task)

    # Generate bead creation commands
    for label, tasks in by_label.items():
        lines.append(f'# === {label.upper()} ===')
        lines.append('')

        for task in tasks:
            # Create the bead
            title = escape_shell(task['name'])
            desc = escape_shell(task['description'][:200]) if task['description'] else ''

            cmd = f'bd create "{title}"'
            cmd += f' -p {task["priority"]}'
            cmd += f' --label {task["label"]}'

            lines.append(f'echo "Creating: {task["id"]}"')
            lines.append(cmd)

            # Capture the bead ID
            lines.append(f'{task["var_name"]}=$(bd list --json 2>/dev/null | jq -r ".[-1].id // empty")')
            lines.append(f'if [ -z "${task["var_name"]}" ]; then')
            lines.append(f'    echo "Warning: Could not capture ID for {task["id"]}"')
            lines.append('fi')
            lines.append('')

    # Add dependencies in a second pass
    lines.append('# === DEPENDENCIES ===')
    lines.append('echo "Adding dependencies..."')
    lines.append('')

    for task in data['tasks']:
        for dep_id in task['dependencies']:
            if dep_id in data['task_id_map']:
                dep_var = data['task_id_map'][dep_id]
                lines.append(f'if [ -n "${task["var_name"]}" ] && [ -n "${dep_var}" ]; then')
                lines.append(f'    bd dep add ${task["var_name"]} ${dep_var} 2>/dev/null || true')
                lines.append('fi')

    # Handle already-completed tasks
    lines.append('')
    lines.append('# === MARK COMPLETED TASKS ===')
    completed = [t for t in data['tasks'] if t['status'] == 'closed']
    if completed:
        lines.append('echo "Marking completed tasks..."')
        for task in completed:
            lines.append(f'if [ -n "${task["var_name"]}" ]; then')
            lines.append(f'    bd update ${task["var_name"]} --status closed 2>/dev/null || true')
            lines.append('fi')

    # Summary
    lines.append('')
    lines.append('echo ""')
    lines.append('echo "=== Migration Complete ==="')
    lines.append(f'echo "Total beads created: {len(data["tasks"])}"')
    lines.append('echo ""')
    lines.append('echo "View your task graph:"')
    lines.append('echo "  bv                    # Interactive TUI"')
    lines.append('echo "  bv --robot-triage     # AI recommendations"')
    lines.append('echo "  bd ready              # List unblocked tasks"')

    return '\n'.join(lines)


def generate_json_mapping(data: dict) -> str:
    """Generate a JSON file mapping old task IDs to bead IDs."""
    mapping = {
        'project': data['project'],
        'source': 'tasks.yaml',
        'tasks': [
            {
                'yaml_id': t['id'],
                'name': t['name'],
                'label': t['label'],
                'status': t['status'],
                'dependencies': t['dependencies']
            }
            for t in data['tasks']
        ]
    }
    return json.dumps(mapping, indent=2)


def run_claude_migration(yaml_path: Path, verify: bool = False) -> int:
    """Fall back to Claude Code for migration."""
    script_dir = Path(__file__).parent
    claude_script = script_dir / 'migrate-with-claude.sh'

    if not claude_script.exists():
        print(f"Error: Claude migration script not found at {claude_script}", file=sys.stderr)
        return 1

    cmd = ['bash', str(claude_script), str(yaml_path)]
    if verify:
        cmd.append('--verify')

    return subprocess.call(cmd)


def main():
    parser = argparse.ArgumentParser(
        description='Convert tasks.yaml to Beads format'
    )
    parser.add_argument('yaml_file', type=Path, help='Path to tasks.yaml')
    parser.add_argument('--execute', action='store_true',
                        help='Execute the generated script')
    parser.add_argument('--dry-run', action='store_true',
                        help='Print commands without executing')
    parser.add_argument('--json', action='store_true',
                        help='Output JSON mapping instead of shell script')
    parser.add_argument('--output', '-o', type=Path,
                        help='Write output to file instead of stdout')
    parser.add_argument('--fallback-claude', action='store_true',
                        help='Use Claude Code if Python parsing fails')
    parser.add_argument('--claude-only', action='store_true',
                        help='Skip Python parsing, use Claude Code directly')
    parser.add_argument('--verify', action='store_true',
                        help='Verify all tasks migrated (with --fallback-claude)')

    args = parser.parse_args()

    if not args.yaml_file.exists():
        print(f"Error: {args.yaml_file} not found", file=sys.stderr)
        sys.exit(1)

    # Claude-only mode
    if args.claude_only:
        print("Using Claude Code for migration...", file=sys.stderr)
        sys.exit(run_claude_migration(args.yaml_file, args.verify))

    # Check YAML availability
    if not YAML_AVAILABLE:
        if args.fallback_claude:
            print("PyYAML not available, falling back to Claude Code...", file=sys.stderr)
            sys.exit(run_claude_migration(args.yaml_file, args.verify))
        else:
            print("Error: PyYAML not installed. Install with: pip install pyyaml", file=sys.stderr)
            print("Or use --fallback-claude to use Claude Code instead", file=sys.stderr)
            sys.exit(1)

    # Try to parse the YAML
    try:
        data = parse_tasks_yaml(args.yaml_file)
    except Exception as e:
        if args.fallback_claude:
            print(f"YAML parsing failed: {e}", file=sys.stderr)
            print("Falling back to Claude Code...", file=sys.stderr)
            sys.exit(run_claude_migration(args.yaml_file, args.verify))
        else:
            print(f"Error parsing YAML: {e}", file=sys.stderr)
            print("Use --fallback-claude to try Claude Code instead", file=sys.stderr)
            sys.exit(1)

    if args.json:
        output = generate_json_mapping(data)
    else:
        output = generate_shell_script(data)

    # Write or print output
    if args.output:
        args.output.write_text(output)
        if not args.json:
            args.output.chmod(0o755)
        print(f"Written to {args.output}", file=sys.stderr)
    elif args.execute:
        # Write to temp file and execute
        import tempfile
        with tempfile.NamedTemporaryFile(mode='w', suffix='.sh', delete=False) as f:
            f.write(output)
            script_path = f.name

        try:
            result = subprocess.run(['bash', script_path], check=False)
            if result.returncode != 0 and args.fallback_claude:
                print("Shell script execution failed, trying Claude Code...", file=sys.stderr)
                sys.exit(run_claude_migration(args.yaml_file, args.verify))
            elif result.returncode != 0:
                sys.exit(result.returncode)
        finally:
            Path(script_path).unlink()
    else:
        print(output)


if __name__ == '__main__':
    main()
