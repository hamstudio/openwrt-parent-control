package xyz.hamguy.parentcontrol.ui.screens

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import kotlinx.coroutines.launch
import xyz.hamguy.parentcontrol.R
import xyz.hamguy.parentcontrol.model.*
import xyz.hamguy.parentcontrol.repository.ParentControlRepository
import xyz.hamguy.parentcontrol.ui.theme.DangerRed
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun MemberEditorDialog(
    member: Member?,
    repository: ParentControlRepository,
    onDismiss: () -> Unit
) {
    val isEditMode = member != null
    val devices by repository.devices.collectAsState()
    val categories by repository.categories.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    var name by remember { mutableStateOf(member?.name ?: "") }
    var avatar by remember { mutableStateOf(member?.avatar ?: "boy") }
    var selectedMacs by remember { mutableStateOf(member?.deviceMacs?.toSet() ?: emptySet()) }
    var quotaMinutes by remember { mutableFloatStateOf((member?.quotaMinutes ?: 120).toFloat()) }

    var scheduleEnabled by remember { mutableStateOf(member?.schedule?.enabled ?: true) }
    var scheduleAction by remember { mutableStateOf(member?.schedule?.action ?: "block") }
    var selectedDays by remember { mutableStateOf(member?.schedule?.days?.toSet() ?: setOf(0, 1, 2, 3, 4, 5, 6)) }
    var timeRanges by remember {
        mutableStateOf(
            member?.schedule?.timeRanges?.takeIf { it.isNotEmpty() }
                ?: listOf(TimeRange("21:30", "07:00"))
        )
    }

    var blockedAppIds by remember { mutableStateOf(member?.blockedAppIds?.toSet() ?: emptySet()) }
    var isSaving by remember { mutableStateOf(false) }
    var showDeleteConfirm by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        if (categories.isEmpty()) {
            repository.fetchAppCategories()
        }
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = {
                        Text(
                            if (isEditMode) stringResource(R.string.edit_member_title) else stringResource(R.string.add_member_title),
                            fontWeight = FontWeight.Bold
                        )
                    },
                    navigationIcon = {
                        IconButton(onClick = onDismiss) {
                            Icon(Icons.Default.Close, contentDescription = stringResource(R.string.cancel))
                        }
                    },
                    actions = {
                        TextButton(
                            onClick = {
                                isSaving = true
                                val finalMember = Member(
                                    id = member?.id ?: "m_${System.currentTimeMillis()}",
                                    name = name.trim(),
                                    avatar = avatar,
                                    deviceMacs = selectedMacs.toList(),
                                    enabled = true,
                                    isLocked = member?.isLocked ?: false,
                                    bonusUntil = member?.bonusUntil,
                                    quotaMinutes = quotaMinutes.toInt(),
                                    usedMinutes = member?.usedMinutes ?: 0,
                                    schedule = ScheduleRule(
                                        enabled = scheduleEnabled && timeRanges.isNotEmpty(),
                                        days = selectedDays.sorted(),
                                        timeRanges = timeRanges,
                                        action = scheduleAction
                                    ),
                                    blockedAppIds = blockedAppIds.toList(),
                                    safeSearch = true,
                                    blockAdult = true
                                )
                                coroutineScope.launch {
                                    repository.saveMember(finalMember)
                                    isSaving = false
                                    onDismiss()
                                }
                            },
                            enabled = name.trim().isNotEmpty() && !isSaving
                        ) {
                            Text(
                                stringResource(R.string.btn_save),
                                fontWeight = FontWeight.Bold,
                                color = if (name.trim().isNotEmpty()) SuccessGreen else Color.Gray
                            )
                        }
                    },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = MaterialTheme.colorScheme.surfaceVariant
                    )
                )
            }
        ) { innerPadding ->
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding)
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp)
            ) {
                // Basic Info Section
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Text(
                                text = stringResource(R.string.member_basic_section),
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold
                            )
                            Spacer(modifier = Modifier.height(12.dp))

                            OutlinedTextField(
                                value = name,
                                onValueChange = { name = it },
                                label = { Text(stringResource(R.string.member_name_placeholder)) },
                                modifier = Modifier.fillMaxWidth(),
                                singleLine = true
                            )

                            Spacer(modifier = Modifier.height(12.dp))
                            Text(
                                text = stringResource(R.string.avatar_label),
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Spacer(modifier = Modifier.height(8.dp))

                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceEvenly
                            ) {
                                val avatars = listOf(
                                    "boy" to "👦",
                                    "girl" to "👧",
                                    "student" to "🧑‍🎓",
                                    "child" to "👶"
                                )
                                avatars.forEach { (key, emoji) ->
                                    val isSelected = avatar == key
                                    Box(
                                        modifier = Modifier
                                            .size(54.dp)
                                            .clip(RoundedCornerShape(14.dp))
                                            .background(
                                                if (isSelected) SuccessGreen.copy(alpha = 0.2f)
                                                else MaterialTheme.colorScheme.surface
                                            )
                                            .clickable { avatar = key },
                                        contentAlignment = Alignment.Center
                                    ) {
                                        Text(text = emoji, fontSize = 28.sp)
                                    }
                                }
                            }
                        }
                    }
                }

                // Device Binding Section
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Text(
                                text = "${stringResource(R.string.device_binding_section)} (${selectedMacs.size})",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold
                            )
                            Spacer(modifier = Modifier.height(8.dp))

                            if (devices.isEmpty()) {
                                Text(
                                    text = stringResource(R.string.no_devices),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            } else {
                                devices.forEach { dev ->
                                    val isChecked = selectedMacs.contains(dev.mac)
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clip(RoundedCornerShape(8.dp))
                                            .clickable {
                                                selectedMacs = if (isChecked) {
                                                    selectedMacs - dev.mac
                                                } else {
                                                    selectedMacs + dev.mac
                                                }
                                            }
                                            .padding(vertical = 6.dp, horizontal = 4.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Checkbox(
                                            checked = isChecked,
                                            onCheckedChange = { checked ->
                                                selectedMacs = if (checked == true) selectedMacs + dev.mac else selectedMacs - dev.mac
                                            }
                                        )
                                        Spacer(modifier = Modifier.width(6.dp))
                                        Column(modifier = Modifier.weight(1f)) {
                                            Text(text = dev.displayName, fontWeight = FontWeight.Medium)
                                            Text(
                                                text = "${dev.ip} · ${dev.mac}",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                        if (dev.online) {
                                            Box(
                                                modifier = Modifier
                                                    .size(8.dp)
                                                    .clip(CircleShape)
                                                    .background(SuccessGreen)
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                // Daily Quota Section
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = stringResource(R.string.daily_quota_section),
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold
                                )
                                Text(
                                    text = if (quotaMinutes <= 0f) stringResource(R.string.unlimited) else "${quotaMinutes.toInt()} ${stringResource(R.string.minutes)}",
                                    color = SuccessGreen,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                            Spacer(modifier = Modifier.height(8.dp))
                            Slider(
                                value = quotaMinutes,
                                onValueChange = { quotaMinutes = it },
                                valueRange = 0f..360f,
                                steps = 23,
                                colors = SliderDefaults.colors(
                                    thumbColor = SuccessGreen,
                                    activeTrackColor = SuccessGreen
                                )
                            )
                        }
                    }
                }

                // Schedule Restrictions Section
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = stringResource(R.string.schedule_section),
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold
                                )
                                Switch(
                                    checked = scheduleEnabled,
                                    onCheckedChange = { scheduleEnabled = it }
                                )
                            }

                            AnimatedVisibility(visible = scheduleEnabled) {
                                Column(modifier = Modifier.padding(top = 12.dp)) {
                                    // Mode Selector (Block vs Allow)
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                                    ) {
                                        FilterChip(
                                            selected = scheduleAction == "block",
                                            onClick = { scheduleAction = "block" },
                                            label = { Text("🚫 " + stringResource(R.string.schedule_action_block)) },
                                            modifier = Modifier.weight(1f)
                                        )
                                        FilterChip(
                                            selected = scheduleAction == "allow",
                                            onClick = { scheduleAction = "allow" },
                                            label = { Text("✅ " + stringResource(R.string.schedule_action_allow)) },
                                            modifier = Modifier.weight(1f)
                                        )
                                    }

                                    Spacer(modifier = Modifier.height(12.dp))
                                    Text(
                                        text = stringResource(R.string.repeat_days),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                    Spacer(modifier = Modifier.height(6.dp))

                                    // Week Days selector
                                    val daysName = listOf("日", "一", "二", "三", "四", "五", "六")
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                                    ) {
                                        (0..6).forEach { day ->
                                            val isSelected = selectedDays.contains(day)
                                            Box(
                                                modifier = Modifier
                                                    .weight(1f)
                                                    .height(36.dp)
                                                    .clip(RoundedCornerShape(8.dp))
                                                    .background(if (isSelected) SuccessGreen else MaterialTheme.colorScheme.surface)
                                                    .clickable {
                                                        selectedDays = if (isSelected) selectedDays - day else selectedDays + day
                                                    },
                                                contentAlignment = Alignment.Center
                                            ) {
                                                Text(
                                                    text = daysName[day],
                                                    color = if (isSelected) Color.White else MaterialTheme.colorScheme.onSurface,
                                                    fontWeight = FontWeight.Bold,
                                                    fontSize = 12.sp
                                                )
                                            }
                                        }
                                    }

                                    Spacer(modifier = Modifier.height(16.dp))
                                    Row(
                                        modifier = Modifier.fillMaxWidth(),
                                        horizontalArrangement = Arrangement.SpaceBetween,
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Text(
                                            text = "时段范围 (${timeRanges.size})",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant
                                        )
                                        TextButton(onClick = {
                                            timeRanges = timeRanges + TimeRange("21:30", "07:00")
                                        }) {
                                            Icon(Icons.Default.Add, contentDescription = null, modifier = Modifier.size(16.dp))
                                            Spacer(modifier = Modifier.width(4.dp))
                                            Text("添加时段")
                                        }
                                    }

                                    timeRanges.forEachIndexed { index, tr ->
                                        Row(
                                            modifier = Modifier
                                                .fillMaxWidth()
                                                .padding(vertical = 4.dp),
                                            verticalAlignment = Alignment.CenterVertically,
                                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                                        ) {
                                            OutlinedTextField(
                                                value = tr.startTime,
                                                onValueChange = { newStart ->
                                                    timeRanges = timeRanges.toMutableList().also {
                                                        it[index] = it[index].copy(startTime = newStart)
                                                    }
                                                },
                                                label = { Text("开始") },
                                                modifier = Modifier.weight(1f),
                                                singleLine = true
                                            )
                                            Text("至", color = MaterialTheme.colorScheme.onSurfaceVariant)
                                            OutlinedTextField(
                                                value = tr.endTime,
                                                onValueChange = { newEnd ->
                                                    timeRanges = timeRanges.toMutableList().also {
                                                        it[index] = it[index].copy(endTime = newEnd)
                                                    }
                                                },
                                                label = { Text("结束") },
                                                modifier = Modifier.weight(1f),
                                                singleLine = true
                                            )
                                            if (timeRanges.size > 1) {
                                                IconButton(onClick = {
                                                    timeRanges = timeRanges.toMutableList().also { it.removeAt(index) }
                                                }) {
                                                    Icon(Icons.Default.Delete, contentDescription = null, tint = DangerRed)
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                // L7 DPI App Restrictions Section
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = stringResource(R.string.blocked_apps_section),
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold
                                )
                                Text(
                                    text = "已选 ${blockedAppIds.size} 项",
                                    color = SuccessGreen,
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                            Spacer(modifier = Modifier.height(12.dp))

                            if (categories.isEmpty()) {
                                Text(
                                    text = "正在加载应用特征库...",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            } else {
                                categories.forEach { cat ->
                                    var isExpanded by remember { mutableStateOf(true) }
                                    val catAppIds = cat.apps.map { it.id }.toSet()
                                    val isAllSelected = catAppIds.isNotEmpty() && catAppIds.all { blockedAppIds.contains(it) }

                                    Column(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(vertical = 4.dp)
                                            .clip(RoundedCornerShape(10.dp))
                                            .background(MaterialTheme.colorScheme.surface)
                                            .padding(10.dp)
                                    ) {
                                        Row(
                                            modifier = Modifier.fillMaxWidth(),
                                            horizontalArrangement = Arrangement.SpaceBetween,
                                            verticalAlignment = Alignment.CenterVertically
                                        ) {
                                            Row(
                                                verticalAlignment = Alignment.CenterVertically,
                                                modifier = Modifier.clickable { isExpanded = !isExpanded }
                                            ) {
                                                Icon(
                                                    imageVector = if (isExpanded) Icons.Default.ExpandMore else Icons.Default.ChevronRight,
                                                    contentDescription = null,
                                                    modifier = Modifier.size(18.dp)
                                                )
                                                Spacer(modifier = Modifier.width(4.dp))
                                                Text(
                                                    text = cat.classZh.ifEmpty { cat.className },
                                                    fontWeight = FontWeight.Bold,
                                                    style = MaterialTheme.typography.titleSmall
                                                )
                                            }

                                            TextButton(onClick = {
                                                blockedAppIds = if (isAllSelected) {
                                                    blockedAppIds - catAppIds
                                                } else {
                                                    blockedAppIds + catAppIds
                                                }
                                            }) {
                                                Text(if (isAllSelected) "取消全选" else "全选", fontSize = 12.sp)
                                            }
                                        }

                                        AnimatedVisibility(visible = isExpanded) {
                                            FlowRow(
                                                modifier = Modifier
                                                    .fillMaxWidth()
                                                    .padding(top = 8.dp),
                                                horizontalArrangement = Arrangement.spacedBy(6.dp),
                                                verticalArrangement = Arrangement.spacedBy(6.dp)
                                            ) {
                                                cat.apps.forEach { app ->
                                                    val isSelected = blockedAppIds.contains(app.id)
                                                    FilterChip(
                                                        selected = isSelected,
                                                        onClick = {
                                                            blockedAppIds = if (isSelected) {
                                                                blockedAppIds - app.id
                                                            } else {
                                                                blockedAppIds + app.id
                                                            }
                                                        },
                                                        label = { Text(app.name, fontSize = 12.sp) },
                                                        leadingIcon = if (isSelected) {
                                                            { Icon(Icons.Default.Check, contentDescription = null, modifier = Modifier.size(14.dp)) }
                                                        } else null
                                                    )
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }

                // Delete Button (Edit mode only)
                if (isEditMode) {
                    item {
                        Button(
                            onClick = { showDeleteConfirm = true },
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.buttonColors(containerColor = DangerRed.copy(alpha = 0.12f)),
                            shape = RoundedCornerShape(12.dp)
                        ) {
                            Text(
                                stringResource(R.string.btn_delete),
                                color = DangerRed,
                                fontWeight = FontWeight.Bold
                            )
                        }
                    }
                }
            }
        }
    }

    if (showDeleteConfirm && member != null) {
        AlertDialog(
            onDismissRequest = { showDeleteConfirm = false },
            title = { Text(stringResource(R.string.btn_delete)) },
            text = { Text(stringResource(R.string.delete_confirm)) },
            confirmButton = {
                Button(
                    onClick = {
                        coroutineScope.launch {
                            repository.deleteMember(member.id)
                            showDeleteConfirm = false
                            onDismiss()
                        }
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = DangerRed)
                ) {
                    Text(stringResource(R.string.btn_delete))
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteConfirm = false }) {
                    Text(stringResource(R.string.cancel))
                }
            }
        )
    }
}
