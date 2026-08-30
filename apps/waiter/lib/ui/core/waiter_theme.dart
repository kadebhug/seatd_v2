import 'package:flutter/material.dart';

ThemeData buildWaiterTheme() {
  const seed = Color(0xff2f6f4e);
  return ThemeData(
    colorScheme: ColorScheme.fromSeed(
      seedColor: seed,
      brightness: Brightness.light,
    ),
    useMaterial3: true,
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        visualDensity: VisualDensity.standard,
        minimumSize: WidgetStateProperty.all(const Size(72, 44)),
      ),
    ),
  );
}
