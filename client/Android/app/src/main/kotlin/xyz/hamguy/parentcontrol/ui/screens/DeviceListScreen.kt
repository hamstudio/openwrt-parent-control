package xyz.hamguy.parentcontrol.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.launch
import xyz.hamguy.parentcontrol.R
import xyz.hamguy.parentcontrol.model.Device
import xyz.hamguy.parentcontrol.model.Member
import xyz.hamguy.parentcontrol.repository.ParentControlRepository
import xyz.hamguy.parentcontrol.ui.theme.DangerRed
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen
import xyz.hamguy.parentcontrol.ui.theme.WarningOrange

enum class AndroidDeviceFilter {
    ALL, ONLINE, LOCKED, UNASSIGNED
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeviceListScreen(
    repository: ParentControlRepository,
    modifier: Modifier = Modifier
) {
    val devices by repository.devices.collectAsState()
    val members by repository.members.collectAsState()
    val needsPinAuth by repository.needsPinAuth.collectAsState()
    val isRefreshing by repository.isLoading.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    var searchQuery by remember { mutableStateOf("") }
    var selectedFilter by remember { mutableStateOf(AndroidDeviceFilter.ALL) }
    var assigningDevice by remember { mutableStateOf<Device?>(null) }
    var pinInput by remember { mutableStateOf("") }

    val filteredDevices = remember(devices, members, searchQuery, selectedFilter) {
        devices.filter { d ->
            val matchesSearch = searchQuery.isEmpty() ||
                    d.displayName.contains(searchQuery, ignoreCase = true) ||
                    d.ip.contains(searchQuery) ||
                    d.mac.contains(searchQuery, ignoreCase = true) ||
                    d.vendor.contains(searchQuery, ignoreCase = true)

            if (!matchesSearch) return@filter false

            when (selectedFilter) {
                AndroidDeviceFilter.ALL -> true
                AndroidDeviceFilter.ONLINE -> d.online
                AndroidDeviceFilter.LOCKED -> d.isLocked
                AndroidDeviceFilter.UNASSIGNED -> {
                    val isAssigned = members.any { m -> m.id == d.memberId || m.deviceMacs.contains(d.mac) }
                    !isAssigned
                }
            }
        }
    }

    val onlineCount = devices.count { it.online }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(stringResource(R.string.devices_title), fontWeight = FontWeight.Bold)
                        Text(
                            text = "$onlineCount / ${devices.size} ${stringResource(R.string.filter_online)}",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                },
                actions = {
                    IconButton(onClick = { coroutineScope.launch { repository.refreshAll() } }) {
                        if (isRefreshing) {
                            CircularProgressIndicator(modifier = Modifier.size(20.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                        }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            )
        }
    ) { innerPadding ->
        Column(
            modifier = modifier
                .fillMaxSize()
                .padding(innerPadding)
        ) {
            // Search field
            OutlinedTextField(
                value = searchQuery,
                onValueChange = { searchQuery = it },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                placeholder = { Text(stringResource(R.string.search_device_placeholder)) },
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                trailingIcon = {
                    if (searchQuery.isNotEmpty()) {
                        IconButton(onClick = { searchQuery = "" }) {
                            Icon(Icons.Default.Close, contentDescription = "Clear")
                        }
                    }
                },
                singleLine = true,
                shape = RoundedCornerShape(12.dp)
            )

            // Filter Tabs
            ScrollableTabRow(
                selectedTabIndex = selectedFilter.ordinal,
                edgePadding = 16.dp,
                divider = {},
                modifier = Modifier.fillMaxWidth()
            ) {
                AndroidDeviceFilter.values().forEach { filter ->
                    val title = when (filter) {
                        AndroidDeviceFilter.ALL -> "${stringResource(R.string.filter_all)} (${devices.size})"
                        AndroidDeviceFilter.ONLINE -> "${stringResource(R.string.filter_online)} ($onlineCount)"
                        AndroidDeviceFilter.LOCKED -> "${stringResource(R.string.filter_locked)} (${devices.count { it.isLocked }})"
                        AndroidDeviceFilter.UNASSIGNED -> stringResource(R.string.filter_unassigned)
                    }
                    Tab(
                        selected = selectedFilter == filter,
                        onClick = { selectedFilter = filter },
                        text = { Text(title, fontWeight = if (selectedFilter == filter) FontWeight.Bold else FontWeight.Normal) }
                    )
                }
            }

            // PIN Auth Card in Devices list if needed
            if (needsPinAuth) {
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    shape = RoundedCornerShape(14.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(Icons.Default.Shield, contentDescription = null, tint = WarningOrange)
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                stringResource(R.string.pin_required_title),
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleSmall
                            )
                        }
                        Text(
                            text = stringResource(R.string.pin_required_desc),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(vertical = 6.dp)
                        )
                        var pinVisible by remember { mutableStateOf(false) }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            OutlinedTextField(
                                value = pinInput,
                                onValueChange = { pinInput = it },
                                placeholder = { Text(stringResource(R.string.pin_placeholder)) },
                                modifier = Modifier.weight(1f),
                                singleLine = true,
                                shape = RoundedCornerShape(10.dp),
                                visualTransformation = if (pinVisible) androidx.compose.ui.text.input.VisualTransformation.None else androidx.compose.ui.text.input.PasswordVisualTransformation(),
                                keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword),
                                trailingIcon = {
                                    IconButton(onClick = { pinVisible = !pinVisible }) {
                                        Icon(
                                            imageVector = if (pinVisible) Icons.Default.VisibilityOff else Icons.Default.Visibility,
                                            contentDescription = null
                                        )
                                    }
                                }
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Button(
                                onClick = {
                                    repository.pinCode = pinInput
                                    coroutineScope.launch { repository.refreshAll() }
                                },
                                colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen),
                                shape = RoundedCornerShape(10.dp)
                            ) {
                                Text(stringResource(R.string.btn_verify), fontWeight = FontWeight.Bold)
                            }
                        }
                    }
                }
            }

            // Devices List
            LazyColumn(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                if (filteredDevices.isEmpty()) {
                    item {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 40.dp),
                            contentAlignment = Alignment.Center
                        ) {
                            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                                Icon(
                                    imageVector = Icons.Default.Devices,
                                    contentDescription = null,
                                    modifier = Modifier.size(48.dp),
                                    tint = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.5f)
                                )
                                Spacer(modifier = Modifier.height(10.dp))
                                Text(
                                    text = stringResource(R.string.no_devices),
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                } else {
                    items(filteredDevices, key = { it.mac }) { device ->
                        val assignedMember = members.firstOrNull { m ->
                            m.id == device.memberId || m.deviceMacs.contains(device.mac)
                        }

                        DeviceItemCard(
                            device = device,
                            assignedMember = assignedMember,
                            onAssignClick = { assigningDevice = device },
                            onToggleLock = {
                                coroutineScope.launch {
                                    repository.lockDevice(device.mac, !device.isLocked)
                                }
                            }
                        )
                    }
                }
            }
        }
    }

    // Assignment Dialog
    assigningDevice?.let { dev ->
        DeviceAssignDialog(
            device = dev,
            members = members,
            onDismiss = { assigningDevice = null },
            onConfirm = { selectedMemberId ->
                coroutineScope.launch {
                    repository.assignDevice(dev.mac, selectedMemberId)
                    assigningDevice = null
                }
            }
        )
    }
}

@Composable
fun DeviceItemCard(
    device: Device,
    assignedMember: Member?,
    onAssignClick: () -> Unit,
    onToggleLock: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Status & Vendor Icon Box
                Box(
                    modifier = Modifier
                        .size(44.dp)
                        .clip(RoundedCornerShape(10.dp))
                        .background(if (device.online) SuccessGreen.copy(alpha = 0.12f) else MaterialTheme.colorScheme.surface),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = when {
                            device.vendor.contains("apple", ignoreCase = true) -> Icons.Default.Laptop
                            device.vendor.contains("sony", ignoreCase = true) || device.vendor.contains("nintendo", ignoreCase = true) -> Icons.Default.VideogameAsset
                            device.hostname.contains("pc", ignoreCase = true) || device.hostname.contains("mac", ignoreCase = true) -> Icons.Default.Computer
                            else -> Icons.Default.Smartphone
                        },
                        contentDescription = null,
                        tint = if (device.online) SuccessGreen else MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                Spacer(modifier = Modifier.width(12.dp))

                // Hostname, IP, MAC
                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(7.dp)
                                .clip(CircleShape)
                                .background(if (device.online) SuccessGreen else Color.Gray)
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = device.displayName,
                            style = MaterialTheme.typography.titleSmall,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1
                        )
                        if (device.isLocked) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Surface(
                                color = DangerRed.copy(alpha = 0.15f),
                                shape = RoundedCornerShape(4.dp)
                            ) {
                                Text(
                                    text = stringResource(R.string.locked),
                                    color = DangerRed,
                                    modifier = Modifier.padding(horizontal = 4.dp, vertical = 1.dp),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontSize = 10.sp,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = "IP: ${device.ip} · MAC: ${device.mac}",
                        style = MaterialTheme.typography.labelSmall,
                        fontFamily = FontFamily.Monospace,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(top = 2.dp)
                    ) {
                        Text(
                            text = device.vendor.ifEmpty { "Generic" },
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        if (device.online) {
                            Text(
                                text = " · ${stringResource(R.string.realtime_speed)}: ${formatSpeed(device.rxRate)}",
                                style = MaterialTheme.typography.labelSmall,
                                color = SuccessGreen,
                                fontWeight = FontWeight.Medium
                            )
                        }
                    }
                }
            }

            HorizontalDivider(
                modifier = Modifier.padding(vertical = 10.dp),
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.08f)
            )

            // Action & Assignment Bar
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                // Member Assignment Badge
                if (assignedMember != null) {
                    Surface(
                        color = SuccessGreen.copy(alpha = 0.12f),
                        shape = RoundedCornerShape(6.dp)
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = when (assignedMember.avatar) {
                                    "girl" -> "👧"
                                    "student" -> "🧑‍🎓"
                                    "child" -> "👶"
                                    else -> "👦"
                                },
                                fontSize = 12.sp
                            )
                            Spacer(modifier = Modifier.width(4.dp))
                            Text(
                                text = assignedMember.name,
                                color = SuccessGreen,
                                style = MaterialTheme.typography.labelSmall,
                                fontWeight = FontWeight.Bold
                            )
                        }
                    }
                } else {
                    Surface(
                        color = MaterialTheme.colorScheme.surface,
                        shape = RoundedCornerShape(6.dp)
                    ) {
                        Text(
                            text = stringResource(R.string.unassigned),
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                            style = MaterialTheme.typography.labelSmall
                        )
                    }
                }

                Row(verticalAlignment = Alignment.CenterVertically) {
                    TextButton(onClick = onAssignClick) {
                        Text(stringResource(R.string.btn_assign), color = SuccessGreen, fontWeight = FontWeight.Bold)
                    }

                    Text(" | ", color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.3f))

                    TextButton(onClick = onToggleLock) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                imageVector = if (device.isLocked) Icons.Default.LockOpen else Icons.Default.Lock,
                                contentDescription = null,
                                modifier = Modifier.size(14.dp),
                                tint = if (device.isLocked) SuccessGreen else DangerRed
                            )
                            Spacer(modifier = Modifier.width(3.dp))
                            Text(
                                text = if (device.isLocked) stringResource(R.string.btn_unlock) else stringResource(R.string.btn_lock),
                                color = if (device.isLocked) SuccessGreen else DangerRed,
                                fontWeight = FontWeight.Bold
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun DeviceAssignDialog(
    device: Device,
    members: List<Member>,
    onDismiss: () -> Unit,
    onConfirm: (String?) -> Unit
) {
    var selectedId by remember { mutableStateOf(device.memberId ?: "") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(stringResource(R.string.assign_modal_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = "${device.displayName} (${device.ip})",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                HorizontalDivider(modifier = Modifier.padding(vertical = 4.dp))

                // Unassigned option
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(8.dp))
                        .background(if (selectedId.isEmpty()) SuccessGreen.copy(alpha = 0.15f) else Color.Transparent)
                        .clickable { selectedId = "" }
                        .padding(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(text = "🚫", fontSize = 18.sp)
                    Spacer(modifier = Modifier.width(10.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Text(stringResource(R.string.unbind_device), fontWeight = FontWeight.Bold)
                        Text(stringResource(R.string.unassigned), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    if (selectedId.isEmpty()) {
                        Icon(Icons.Default.Check, contentDescription = null, tint = SuccessGreen)
                    }
                }

                // Member options
                members.forEach { member ->
                    val isSelected = selectedId == member.id
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .background(if (isSelected) SuccessGreen.copy(alpha = 0.15f) else Color.Transparent)
                            .clickable { selectedId = member.id }
                            .padding(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = when (member.avatar) {
                                "girl" -> "👧"
                                "student" -> "🧑‍🎓"
                                "child" -> "👶"
                                else -> "👦"
                            },
                            fontSize = 18.sp
                        )
                        Spacer(modifier = Modifier.width(10.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(text = member.name, fontWeight = FontWeight.Bold)
                            Text(
                                text = "${member.deviceMacs.size} ${stringResource(R.string.stat_devices)} · ${stringResource(R.string.today_usage)}: ${member.usedMinutes} ${stringResource(R.string.minutes)}",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                        if (isSelected) {
                            Icon(Icons.Default.Check, contentDescription = null, tint = SuccessGreen)
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = { onConfirm(selectedId.ifEmpty { null }) },
                colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen)
            ) {
                Text(stringResource(R.string.btn_save))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(stringResource(R.string.cancel))
            }
        }
    )
}

private fun formatSpeed(bytesPerSec: Long): String {
    return if (bytesPerSec >= 1024 * 1024) {
        String.format("%.1f MB/s", bytesPerSec / (1024.0 * 1024.0))
    } else {
        String.format("%.0f KB/s", bytesPerSec / 1024.0)
    }
}
