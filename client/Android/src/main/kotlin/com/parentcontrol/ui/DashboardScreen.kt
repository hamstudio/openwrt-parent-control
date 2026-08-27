package com.parentcontrol.ui

// Jetpack Compose UI 示例代码展示（直连控制台卡片、一键断网、配额进度）
// 在 Android 客户端中，UI 状态直接通过 ParentControlRepository 消费由 Swift Core 导出的数据。

/*
@Composable
fun DashboardScreen(
    repository: ParentControlRepository,
    viewModel: DashboardViewModel = viewModel()
) {
    val status by repository.statusFlow.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("家长控制中心 (Android)") },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.primaryContainer
                )
            )
        }
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            item {
                StatusCard(status = status)
            }
            // 渲染成员卡片与快捷操作...
        }
    }
}
*/
