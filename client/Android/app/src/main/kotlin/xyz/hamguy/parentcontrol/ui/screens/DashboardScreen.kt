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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.launch
import xyz.hamguy.parentcontrol.R
import xyz.hamguy.parentcontrol.model.Member
import xyz.hamguy.parentcontrol.model.SystemStatus
import xyz.hamguy.parentcontrol.repository.ParentControlRepository
import xyz.hamguy.parentcontrol.ui.theme.DangerRed
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen
import xyz.hamguy.parentcontrol.ui.theme.WarningOrange

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    repository: ParentControlRepository,
    modifier: Modifier = Modifier
) {
    val status by repository.status.collectAsState()
    val members by repository.members.collectAsState()
    val isConnected by repository.isConnected.collectAsState()
    val needsPinAuth by repository.needsPinAuth.collectAsState()
    val isLoading by repository.isLoading.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    var editingMember by remember { mutableStateOf<Member?>(null) }
    var isAddingMember by remember { mutableStateOf(false) }
    var bonusTargetMember by remember { mutableStateOf<Member?>(null) }
    var pinInput by remember { mutableStateOf("") }

    LaunchedEffect(Unit) {
        repository.refreshAll()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.dashboard_title), fontWeight = FontWeight.Bold) },
                actions = {
                    IconButton(onClick = { isAddingMember = true }) {
                        Icon(Icons.Default.PersonAdd, contentDescription = stringResource(R.string.add_member_btn), tint = SuccessGreen)
                    }
                    IconButton(
                        onClick = {
                            coroutineScope.launch { repository.refreshAll() }
                        }
                    ) {
                        if (isLoading) {
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
        LazyColumn(
            modifier = modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            // Status Header & Metric Tiles
            item {
                StatusHeader(
                    status = status,
                    isConnected = isConnected,
                    serverUrl = repository.baseUrl,
                    membersCount = members.size
                )
            }

            // PIN Auth Card (when required)
            if (needsPinAuth) {
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
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
            }

            // Family Members Section Title
            item {
                Text(
                    text = stringResource(R.string.tab_members),
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }

            // Empty State or Member List
            if (members.isEmpty() && !needsPinAuth) {
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(28.dp),
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Icon(
                                imageVector = Icons.Default.Shield,
                                contentDescription = null,
                                modifier = Modifier.size(52.dp),
                                tint = SuccessGreen.copy(alpha = 0.8f)
                            )
                            Spacer(modifier = Modifier.height(12.dp))
                            Text(
                                text = stringResource(R.string.empty_members_title),
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold
                            )
                            Spacer(modifier = Modifier.height(6.dp))
                            Text(
                                text = stringResource(R.string.empty_members_desc),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Spacer(modifier = Modifier.height(16.dp))
                            Button(
                                onClick = { isAddingMember = true },
                                colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen),
                                shape = RoundedCornerShape(12.dp)
                            ) {
                                Icon(Icons.Default.Add, contentDescription = null)
                                Spacer(modifier = Modifier.width(6.dp))
                                Text(stringResource(R.string.add_member_btn), fontWeight = FontWeight.Bold)
                            }
                        }
                    }
                }
            } else {
                items(members, key = { it.id }) { member ->
                    MemberCard(
                        member = member,
                        onEditClick = { editingMember = member },
                        onToggleLock = {
                            coroutineScope.launch {
                                repository.lockMember(member.id, !member.isLocked)
                            }
                        },
                        onBonusClick = {
                            bonusTargetMember = member
                        }
                    )
                }
            }
        }
    }

    // Add or Edit Member Dialog
    if (isAddingMember) {
        MemberEditorDialog(
            member = null,
            repository = repository,
            onDismiss = { isAddingMember = false }
        )
    }

    editingMember?.let { m ->
        MemberEditorDialog(
            member = m,
            repository = repository,
            onDismiss = { editingMember = null }
        )
    }

    // Bonus Time Dialog
    bonusTargetMember?.let { m ->
        AlertDialog(
            onDismissRequest = { bonusTargetMember = null },
            title = { Text("${stringResource(R.string.bonus_modal_title)} - ${m.name}") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    listOf(
                        15 to "+15 分钟",
                        30 to "+30 分钟",
                        60 to "+1 小时",
                        120 to "+2 小时"
                    ).forEach { (mins, label) ->
                        Button(
                            onClick = {
                                coroutineScope.launch {
                                    repository.bonusMember(m.id, mins)
                                    bonusTargetMember = null
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
                            shape = RoundedCornerShape(10.dp)
                        ) {
                            Text(label, color = MaterialTheme.colorScheme.onSurfaceVariant, fontWeight = FontWeight.Bold)
                        }
                    }
                }
            },
            confirmButton = {},
            dismissButton = {
                TextButton(onClick = { bonusTargetMember = null }) {
                    Text(stringResource(R.string.cancel))
                }
            }
        )
    }
}

@Composable
fun StatusHeader(
    status: SystemStatus?,
    isConnected: Boolean,
    serverUrl: String,
    membersCount: Int
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            // Connection Status and DPI Ready pill
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        modifier = Modifier
                            .size(8.dp)
                            .clip(CircleShape)
                            .background(if (isConnected) SuccessGreen else DangerRed)
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        text = if (isConnected) "${stringResource(R.string.connected_to)} (${serverUrl.replace("http://", "")})" else stringResource(R.string.not_connected),
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Bold,
                        color = if (isConnected) MaterialTheme.colorScheme.onSurfaceVariant else DangerRed
                    )
                }

                if (status?.kernelDpiReady == true) {
                    Surface(
                        color = SuccessGreen.copy(alpha = 0.15f),
                        shape = RoundedCornerShape(6.dp)
                    ) {
                        Text(
                            text = "✓ " + stringResource(R.string.dpi_ready),
                            color = SuccessGreen,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                            style = MaterialTheme.typography.labelSmall,
                            fontSize = 11.sp,
                            fontWeight = FontWeight.Bold
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // 3 Metric Tiles
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                MetricTile(
                    title = stringResource(R.string.stat_members),
                    value = "$membersCount",
                    icon = Icons.Default.People,
                    tint = SuccessGreen,
                    modifier = Modifier.weight(1f)
                )
                MetricTile(
                    title = stringResource(R.string.stat_devices),
                    value = "${status?.activeDevices ?: 0} / ${status?.totalDevices ?: 0}",
                    icon = Icons.Default.Devices,
                    tint = Color(0xFF00897B),
                    modifier = Modifier.weight(1f)
                )
                MetricTile(
                    title = stringResource(R.string.stat_apps),
                    value = "${status?.appCount ?: 0}",
                    icon = Icons.Default.GridView,
                    tint = Color(0xFF0288D1),
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
fun MetricTile(
    title: String,
    value: String,
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    tint: Color,
    modifier: Modifier = Modifier
) {
    Surface(
        modifier = modifier,
        shape = RoundedCornerShape(10.dp),
        color = MaterialTheme.colorScheme.surface
    ) {
        Column(modifier = Modifier.padding(10.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    fontSize = 11.sp
                )
                Icon(
                    imageVector = icon,
                    contentDescription = null,
                    modifier = Modifier.size(14.dp),
                    tint = tint
                )
            }
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = value,
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold
            )
        }
    }
}

@Composable
fun MemberCard(
    member: Member,
    onEditClick: () -> Unit,
    onToggleLock: () -> Unit,
    onBonusClick: () -> Unit
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(16.dp),
        colors = CardDefaults.cardColors(
            containerColor = if (member.isLocked) MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.3f)
            else MaterialTheme.colorScheme.surfaceVariant
        )
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            // Header: Avatar, Name, Status Badge, Edit Button
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Box(
                    modifier = Modifier
                        .size(46.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(SuccessGreen.copy(alpha = 0.12f)),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = when (member.avatar) {
                            "girl" -> "👧"
                            "student" -> "🧑‍🎓"
                            "child" -> "👶"
                            else -> "👦"
                        },
                        fontSize = 24.sp
                    )
                }

                Spacer(modifier = Modifier.width(12.dp))

                Column(modifier = Modifier.weight(1f)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            text = member.name,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        StatusBadge(member = member)
                    }
                    Text(
                        text = "${member.deviceMacs.size} ${stringResource(R.string.stat_devices)} · ${member.blockedAppIds.size} ${stringResource(R.string.stat_apps)}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                IconButton(
                    onClick = onEditClick,
                    modifier = Modifier
                        .size(36.dp)
                        .clip(CircleShape)
                        .background(SuccessGreen.copy(alpha = 0.1f))
                ) {
                    Icon(
                        imageVector = Icons.Default.Tune,
                        contentDescription = stringResource(R.string.btn_edit),
                        tint = SuccessGreen,
                        modifier = Modifier.size(18.dp)
                    )
                }
            }

            // Schedule Summary (if configured)
            if (member.schedule.enabled && member.schedule.timeRanges.isNotEmpty()) {
                Spacer(modifier = Modifier.height(10.dp))
                val isBlock = member.schedule.action == "block"
                val rangesText = member.schedule.timeRanges.joinToString(", ") { "${it.startTime}~${it.endTime}" }
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = SuccessGreen.copy(alpha = 0.08f)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 8.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Icon(
                            imageVector = Icons.Default.Schedule,
                            contentDescription = null,
                            tint = SuccessGreen,
                            modifier = Modifier.size(14.dp)
                        )
                        Spacer(modifier = Modifier.width(6.dp))
                        Text(
                            text = "${if (isBlock) "🚫 限制上网" else "✅ 仅允许上网"}: $rangesText",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(10.dp))

            // Quota Bar
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween
            ) {
                Text(
                    stringResource(R.string.today_usage),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = "${member.usedMinutes} / ${if (member.quotaMinutes > 0) "${member.quotaMinutes} ${stringResource(R.string.minutes)}" else stringResource(R.string.unlimited)}",
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.Bold
                )
            }
            Spacer(modifier = Modifier.height(6.dp))
            LinearProgressIndicator(
                progress = { member.quotaProgress },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(8.dp)
                    .clip(RoundedCornerShape(4.dp)),
                color = if (member.quotaProgress > 0.9f) DangerRed else if (member.quotaProgress > 0.7f) WarningOrange else SuccessGreen,
                trackColor = MaterialTheme.colorScheme.surface
            )

            Spacer(modifier = Modifier.height(14.dp))

            // Action Buttons
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                Button(
                    onClick = onToggleLock,
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = if (member.isLocked) SuccessGreen else DangerRed
                    ),
                    shape = RoundedCornerShape(12.dp)
                ) {
                    Icon(
                        imageVector = if (member.isLocked) Icons.Default.LockOpen else Icons.Default.Lock,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp)
                    )
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(if (member.isLocked) stringResource(R.string.btn_unlock) else stringResource(R.string.btn_lock), fontWeight = FontWeight.Bold)
                }

                Button(
                    onClick = onBonusClick,
                    modifier = Modifier.weight(1f),
                    colors = ButtonDefaults.buttonColors(containerColor = WarningOrange),
                    shape = RoundedCornerShape(12.dp)
                ) {
                    Icon(Icons.Default.AddCircle, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(modifier = Modifier.width(4.dp))
                    Text(stringResource(R.string.btn_bonus), fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
fun StatusBadge(member: Member) {
    val (label, bg, fg) = when {
        member.isLocked -> Triple(stringResource(R.string.locked), DangerRed.copy(alpha = 0.15f), DangerRed)
        member.isBonusActive -> Triple(stringResource(R.string.bonus_active), WarningOrange.copy(alpha = 0.15f), WarningOrange)
        member.isQuotaExceeded -> Triple(stringResource(R.string.locked), WarningOrange.copy(alpha = 0.15f), WarningOrange)
        else -> Triple(stringResource(R.string.normal_online), SuccessGreen.copy(alpha = 0.15f), SuccessGreen)
    }

    Surface(
        color = bg,
        shape = RoundedCornerShape(4.dp)
    ) {
        Text(
            text = label,
            color = fg,
            modifier = Modifier.padding(horizontal = 5.dp, vertical = 1.dp),
            style = MaterialTheme.typography.labelSmall,
            fontSize = 10.sp,
            fontWeight = FontWeight.Bold
        )
    }
}
