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
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen

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
    val isRefreshing by repository.isLoading.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    var searchQuery by remember { mutableStateOf("") }
    var selectedFilter by remember { mutableStateOf(AndroidDeviceFilter.ALL) }
    var assigningDevice by remember { mutableStateOf<Device?>(null) }

    val filteredDevices = remember(devices, members, searchQuery, selectedFilter) {
        devices.filter { d ->
            val matchesSearch = searchQuery.isEmpty() ||
                    d.hostname.contains(searchQuery, ignoreCase = true) ||
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
                            text = "$onlineCount / ${devices.size} Online",
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
                placeholder = { Text("Search hostname, IP, MAC, vendor...") },
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
                        AndroidDeviceFilter.ALL -> "All (${devices.size})"
                        AndroidDeviceFilter.ONLINE -> "Online ($onlineCount)"
                        AndroidDeviceFilter.LOCKED -> "Locked (${devices.count { it.isLocked }})"
                        AndroidDeviceFilter.UNASSIGNED -> "Unassigned"
                    }
                    Tab(
                        selected = selectedFilter == filter,
                        onClick = { selectedFilter = filter },
                        text = { Text(title, fontWeight = if (selectedFilter == filter) FontWeight.Bold else FontWeight.Normal) }
                    )
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
                            Text(
                                text = "No matching devices found",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
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
    onAssignClick: () -> VoidHandler,
    onToggleLock: () -> VoidHandler
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
                // Status & Icon
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
                            fontWeight = FontWeight.Bold
                        )
                        if (device.isLocked) {
                            Spacer(modifier = Modifier.width(6.dp))
                            Surface(
                                color = MaterialTheme.colorScheme.error.copy(alpha = 0.15f),
                                shape = RoundedCornerShape(4.dp)
                            ) {
                                Text(
                                    text = "Locked",
                                    color = MaterialTheme.colorScheme.error,
                                    modifier = Modifier.padding(horizontal = 4.dp, vertical = 1.dp),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontSize = 10.sp,
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }

                    Text(
                        text = "IP: ${device.ip}  •  MAC: ${device.mac}",
                        style = MaterialTheme.typography.bodySmall,
                        fontFamily = FontFamily.Monospace,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Row(modifier = Modifier.padding(top = 2.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = device.vendor.ifEmpty { "Generic" },
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        if (device.online) {
                            Text(
                                text = "  •  Rate: ${formatSpeed(device.rxRate)}",
                                style = MaterialTheme.typography.labelSmall,
                                color = SuccessGreen,
                                fontWeight = FontWeight.SemiBold
                            )
                        }
                    }
                }
            }

            Divider(modifier = Modifier.padding(vertical = 10.dp), color = MaterialTheme.colorScheme.surface)

            // Actions & Assignment
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // Member badge
                if (assignedMember != null) {
                    Surface(
                        color = SuccessGreen.copy(alpha = 0.12f),
                        shape = RoundedCornerShape(6.dp)
                    ) {
                        Row(
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
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
                                style = MaterialTheme.typography.labelMedium,
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
                            text = "Unassigned",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                            style = MaterialTheme.typography.labelSmall
                        )
                    }
                }

                Spacer(modifier = Modifier.weight(1f))

                // Assign button
                TextButton(onClick = { onAssignClick() }, contentPadding = PaddingValues(horizontal = 8.dp, vertical = 0.dp)) {
                    Text("Assign", style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold, color = SuccessGreen)
                }

                Text(" | ", color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f), fontSize = 12.sp)

                // Lock toggle button
                TextButton(
                    onClick = { onToggleLock() },
                    contentPadding = PaddingValues(horizontal = 8.dp, vertical = 0.dp)
                ) {
                    Icon(
                        imageVector = if (device.isLocked) Icons.Default.LockOpen else Icons.Default.Lock,
                        contentDescription = null,
                        modifier = Modifier.size(14.dp),
                        tint = if (device.isLocked) SuccessGreen else MaterialTheme.colorScheme.error
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(
                        text = if (device.isLocked) "Unlock" else "Lock",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Bold,
                        color = if (device.isLocked) SuccessGreen else MaterialTheme.colorScheme.error
                    )
                }
            }
        }
    }
}

@Composable
fun DeviceAssignDialog(
    device: Device,
    members: List<Member>,
    onDismiss: () -> VoidHandler,
    onConfirm: (String?) -> VoidHandler
) {
    var selectedId by remember { mutableStateOf(device.memberId ?: "") }

    AlertDialog(
        onDismissRequest = { onDismiss() },
        title = { Text("Assign Device") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = "${device.displayName} (${device.ip})",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Divider(modifier = Modifier.padding(vertical = 4.dp))

                // Unassigned Option
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(8.dp))
                        .clickable { selectedId = "" }
                        .padding(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    RadioButton(selected = selectedId.isEmpty(), onClick = { selectedId = "" })
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("🚫 Unbind (Set as Unassigned)", style = MaterialTheme.typography.bodyMedium)
                }

                // Member options
                members.forEach { m ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .clickable { selectedId = m.id }
                            .padding(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        RadioButton(selected = selectedId == m.id, onClick = { selectedId = m.id })
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "${m.name} (${m.deviceMacs.size} devices)",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }
            }
        },
        confirmButton = {
            Button(onClick = { onConfirm(selectedId.ifEmpty { null }) }) {
                Text("Save")
            }
        },
        dismissButton = {
            TextButton(onClick = { onDismiss() }) {
                Text("Cancel")
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

typealias VoidHandler = () -> Unit
