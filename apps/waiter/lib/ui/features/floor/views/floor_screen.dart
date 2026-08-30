import 'package:flutter/material.dart';
import 'package:seatd_client/seatd_client.dart';

import '../../../../domain/waiter_models.dart';
import '../view_models/floor_view_model.dart';

class FloorScreen extends StatefulWidget {
  const FloorScreen({super.key, required this.viewModel});

  final FloorViewModel viewModel;

  @override
  State<FloorScreen> createState() => _FloorScreenState();
}

class _FloorScreenState extends State<FloorScreen> {
  @override
  void initState() {
    super.initState();
    widget.viewModel.bootstrap();
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: widget.viewModel,
      builder: (context, _) {
        final model = widget.viewModel;
        return Scaffold(
          appBar: AppBar(
            title: const Text('Seatd Waiter'),
            actions: [
              IconButton(
                tooltip: 'Sync',
                onPressed: model.bootstrap,
                icon: const Icon(Icons.sync),
              ),
            ],
          ),
          body: SafeArea(
            child: Column(
              children: [
                _SyncBanner(model: model),
                _FloorFilters(model: model),
                const Divider(height: 1),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      final wide = constraints.maxWidth >= 720;
                      final content = _TableBoard(model: model);
                      if (!wide) {
                        return content;
                      }
                      return Row(
                        children: [
                          SizedBox(
                            width: 260,
                            child: _AssistRail(model: model),
                          ),
                          const VerticalDivider(width: 1),
                          Expanded(child: content),
                        ],
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _SyncBanner extends StatelessWidget {
  const _SyncBanner({required this.model});

  final FloorViewModel model;

  @override
  Widget build(BuildContext context) {
    final state = model.state;
    final scheme = Theme.of(context).colorScheme;
    final lastSync = state.lastSuccessfulSync;
    final message =
        model.statusMessage ??
        (state.isOnline
            ? 'Online${lastSync == null ? '' : ' - synced ${_formatTime(lastSync)}'}'
            : 'Offline - actions will retry');
    return Container(
      width: double.infinity,
      color: state.isOnline ? scheme.primaryContainer : scheme.errorContainer,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          Icon(
            state.isOnline ? Icons.cloud_done : Icons.cloud_off,
            size: 20,
            color: state.isOnline
                ? scheme.onPrimaryContainer
                : scheme.onErrorContainer,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              state.isSyncing ? 'Syncing floor state...' : message,
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (state.commands.any(
            (command) => command.state != CommandState.complete,
          ))
            Text(
              '${state.commands.where((command) => command.state != CommandState.complete).length} pending',
            ),
        ],
      ),
    );
  }
}

class _FloorFilters extends StatelessWidget {
  const _FloorFilters({required this.model});

  final FloorViewModel model;

  @override
  Widget build(BuildContext context) {
    final floors = model.state.floors;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (floors.isNotEmpty)
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: SegmentedButton<String>(
                segments: [
                  for (final floor in floors)
                    ButtonSegment<String>(
                      value: floor.id,
                      icon: const Icon(Icons.layers),
                      label: Text(floor.name),
                    ),
                ],
                selected: {model.selectedFloorId ?? floors.first.id},
                onSelectionChanged: (selection) =>
                    model.selectFloor(selection.first),
              ),
            ),
          const SizedBox(height: 10),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: SegmentedButton<String?>(
              segments: [
                const ButtonSegment<String?>(
                  value: null,
                  icon: Icon(Icons.grid_view),
                  label: Text('All zones'),
                ),
                for (final zone in model.visibleZones)
                  ButtonSegment<String?>(
                    value: zone.id,
                    icon: const Icon(Icons.table_bar),
                    label: Text(zone.name),
                  ),
              ],
              selected: {model.selectedZoneId},
              onSelectionChanged: (selection) =>
                  model.selectZone(selection.first),
            ),
          ),
        ],
      ),
    );
  }
}

class _TableBoard extends StatelessWidget {
  const _TableBoard({required this.model});

  final FloorViewModel model;

  @override
  Widget build(BuildContext context) {
    final tables = model.visibleTables;
    if (tables.isEmpty) {
      return const Center(child: Text('No tables for this floor'));
    }
    return GridView.builder(
      padding: const EdgeInsets.all(12),
      gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
        maxCrossAxisExtent: 220,
        mainAxisExtent: 208,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
      ),
      itemCount: tables.length,
      itemBuilder: (context, index) {
        final table = tables[index];
        return _TableTile(
          table: table,
          assists: model.assistsForTable(table.table.id),
          command: model.commandForEntity(table.table.id),
          onPressed: () => model.toggleTable(table),
          onAssistAction: (assist, action) {
            switch (action) {
              case _AssistAction.acknowledge:
                model.acknowledgeAssist(assist);
              case _AssistAction.resolve:
                model.resolveAssist(assist);
              case _AssistAction.cancel:
                model.cancelAssist(assist);
            }
          },
        );
      },
    );
  }
}

class _TableTile extends StatelessWidget {
  const _TableTile({
    required this.table,
    required this.assists,
    required this.command,
    required this.onPressed,
    required this.onAssistAction,
  });

  final TableState table;
  final List<Assist> assists;
  final WaiterCommand? command;
  final VoidCallback onPressed;
  final void Function(Assist assist, _AssistAction action) onAssistAction;

  @override
  Widget build(BuildContext context) {
    final occupied = table.occupancy.status == 'occupied';
    final scheme = Theme.of(context).colorScheme;
    final color = occupied
        ? scheme.tertiaryContainer
        : scheme.secondaryContainer;
    return Material(
      color: color,
      borderRadius: BorderRadius.circular(8),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onPressed,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  _ShapeIcon(shape: table.table.shape),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      table.table.label,
                      style: Theme.of(context).textTheme.headlineSmall,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  if (assists.isNotEmpty)
                    Badge(
                      label: Text('${assists.length}'),
                      child: const Icon(Icons.notifications_active),
                    ),
                ],
              ),
              const Spacer(),
              Text(occupied ? 'Occupied' : 'Available'),
              Text('Seats ${table.table.capacityLabel}'),
              if (command != null)
                Text(
                  command!.state == CommandState.conflict
                      ? 'Conflict'
                      : 'Queued',
                  style: TextStyle(color: scheme.error),
                ),
              if (assists.isNotEmpty)
                Wrap(
                  spacing: 6,
                  children: [
                    for (final assist in assists.take(1))
                      _AssistButtons(assist: assist, onAction: onAssistAction),
                  ],
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ShapeIcon extends StatelessWidget {
  const _ShapeIcon({required this.shape});

  final String shape;

  @override
  Widget build(BuildContext context) {
    return Icon(switch (shape) {
      'circle' => Icons.circle_outlined,
      'square' => Icons.crop_square,
      _ => Icons.table_restaurant,
    });
  }
}

enum _AssistAction { acknowledge, resolve, cancel }

class _AssistButtons extends StatelessWidget {
  const _AssistButtons({required this.assist, required this.onAction});

  final Assist assist;
  final void Function(Assist assist, _AssistAction action) onAction;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 0,
      runSpacing: 0,
      children: [
        IconButton(
          tooltip: 'Acknowledge assist',
          constraints: const BoxConstraints.tightFor(width: 36, height: 36),
          iconSize: 20,
          padding: EdgeInsets.zero,
          onPressed: assist.status == 'pending'
              ? () => onAction(assist, _AssistAction.acknowledge)
              : null,
          icon: const Icon(Icons.pan_tool_alt),
        ),
        IconButton(
          tooltip: 'Resolve assist',
          constraints: const BoxConstraints.tightFor(width: 36, height: 36),
          iconSize: 20,
          padding: EdgeInsets.zero,
          onPressed: () => onAction(assist, _AssistAction.resolve),
          icon: const Icon(Icons.check_circle),
        ),
        IconButton(
          tooltip: 'Cancel assist',
          constraints: const BoxConstraints.tightFor(width: 36, height: 36),
          iconSize: 20,
          padding: EdgeInsets.zero,
          onPressed: () => onAction(assist, _AssistAction.cancel),
          icon: const Icon(Icons.cancel),
        ),
      ],
    );
  }
}

class _AssistRail extends StatelessWidget {
  const _AssistRail({required this.model});

  final FloorViewModel model;

  @override
  Widget build(BuildContext context) {
    final assists = model.state.assists;
    return ListView(
      padding: const EdgeInsets.all(12),
      children: [
        Text('Assists', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        if (assists.isEmpty) const Text('No active assists'),
        for (final assist in assists)
          ListTile(
            leading: const Icon(Icons.notifications_active),
            title: Text(_tableLabel(model.state.tables, assist.tableId)),
            subtitle: Text(assist.note ?? assist.status),
            trailing: IconButton(
              tooltip: 'Resolve assist',
              onPressed: () => model.resolveAssist(assist),
              icon: const Icon(Icons.check),
            ),
          ),
      ],
    );
  }
}

String _tableLabel(List<TableState> tables, String tableId) {
  for (final table in tables) {
    if (table.table.id == tableId) {
      return table.table.label;
    }
  }
  return 'Table';
}

String _formatTime(DateTime time) {
  final local = time.toLocal();
  return '${local.hour.toString().padLeft(2, '0')}:${local.minute.toString().padLeft(2, '0')}';
}
