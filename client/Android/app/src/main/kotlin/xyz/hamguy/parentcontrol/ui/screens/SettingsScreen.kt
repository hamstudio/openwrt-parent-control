package xyz.hamguy.parentcontrol.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material.icons.filled.WifiFind
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import kotlinx.coroutines.launch
import xyz.hamguy.parentcontrol.R
import xyz.hamguy.parentcontrol.model.GlobalSettings
import xyz.hamguy.parentcontrol.repository.ParentControlRepository
import xyz.hamguy.parentcontrol.ui.theme.DangerRed
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    repository: ParentControlRepository,
    modifier: Modifier = Modifier
) {
    val status by repository.status.collectAsState()
    val settings by repository.settings.collectAsState()
    val isConnected by repository.isConnected.collectAsState()
    val needsPinAuth by repository.needsPinAuth.collectAsState()
    val errorMessage by repository.errorMessage.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    var routerUrl by remember { mutableStateOf(repository.baseUrl) }
    var pinCode by remember { mutableStateOf(repository.pinCode) }
    var isDiscovering by remember { mutableStateOf(false) }
    var isSavingPolicy by remember { mutableStateOf(false) }
    var saveSuccessMsg by remember { mutableStateOf<String?>(null) }

    // Policy States
    var globalEnabled by remember { mutableStateOf(settings.enabled) }
    var safeSearch by remember { mutableStateOf(settings.enforceSafeSearch) }
    var blockDoH by remember { mutableStateOf(settings.blockDoHDoT) }
    var isolateNew by remember { mutableStateOf(settings.isolateNewDevices) }

    LaunchedEffect(settings) {
        globalEnabled = settings.enabled
        safeSearch = settings.enforceSafeSearch
        blockDoH = settings.blockDoHDoT
        isolateNew = settings.isolateNewDevices
    }

    LaunchedEffect(Unit) {
        repository.fetchSettings()
    }

    val candidates = listOf(
        "http://192.168.0.110:8088",
        "http://192.168.1.1:8088",
        "http://192.168.0.1:8088",
        "http://192.168.31.1:8088",
        "http://10.0.0.1:8088"
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.settings_title), fontWeight = FontWeight.Bold) },
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
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Router Connection Section
            item {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            text = stringResource(R.string.router_conn_section),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                        Spacer(modifier = Modifier.height(12.dp))

                        // Router URL & Connect Button
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            OutlinedTextField(
                                value = routerUrl,
                                onValueChange = { routerUrl = it },
                                label = { Text(stringResource(R.string.router_url_label)) },
                                modifier = Modifier.weight(1f),
                                singleLine = true,
                                shape = RoundedCornerShape(10.dp)
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Button(
                                onClick = {
                                    repository.baseUrl = routerUrl
                                    coroutineScope.launch {
                                        repository.refreshAll()
                                        repository.fetchSettings()
                                    }
                                },
                                colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen),
                                shape = RoundedCornerShape(10.dp)
                            ) {
                                Text(stringResource(R.string.btn_connect), fontWeight = FontWeight.Bold)
                            }
                        }

                        Spacer(modifier = Modifier.height(10.dp))

                        // PIN Code input
                        var pinVisible by remember { mutableStateOf(false) }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            OutlinedTextField(
                                value = pinCode,
                                onValueChange = { pinCode = it },
                                label = { Text(stringResource(R.string.pin_section)) },
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
                                    repository.pinCode = pinCode
                                    saveSuccessMsg = "PIN已保存并重新连接"
                                    coroutineScope.launch {
                                        repository.refreshAll()
                                        repository.fetchSettings()
                                    }
                                },
                                colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen),
                                shape = RoundedCornerShape(10.dp)
                            ) {
                                Text(stringResource(R.string.btn_save_pin), fontWeight = FontWeight.Bold)
                            }
                        }

                        Spacer(modifier = Modifier.height(10.dp))

                        // Connection Status Pill
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.padding(vertical = 4.dp)
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(8.dp)
                                    .clip(CircleShape)
                                    .background(if (isConnected && !needsPinAuth) SuccessGreen else DangerRed)
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                text = when {
                                    isConnected && !needsPinAuth -> "${stringResource(R.string.connected_to)} (${repository.baseUrl})"
                                    isConnected && needsPinAuth -> stringResource(R.string.pin_required_title)
                                    else -> errorMessage ?: stringResource(R.string.not_connected)
                                },
                                style = MaterialTheme.typography.bodySmall,
                                color = if (isConnected && !needsPinAuth) SuccessGreen else DangerRed,
                                fontWeight = FontWeight.Medium
                            )
                        }

                        Spacer(modifier = Modifier.height(8.dp))

                        // Auto-Discovery Button
                        OutlinedButton(
                            onClick = {
                                isDiscovering = true
                                coroutineScope.launch {
                                    val found = repository.autoDiscover()
                                    if (found != null) {
                                        routerUrl = found
                                    }
                                    isDiscovering = false
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                            shape = RoundedCornerShape(10.dp)
                        ) {
                            if (isDiscovering) {
                                CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(stringResource(R.string.discovering))
                            } else {
                                Icon(Icons.Default.WifiFind, contentDescription = null, tint = SuccessGreen)
                                Spacer(modifier = Modifier.width(8.dp))
                                Text(stringResource(R.string.btn_auto_discover), color = SuccessGreen, fontWeight = FontWeight.Bold)
                            }
                        }

                        Spacer(modifier = Modifier.height(10.dp))

                        // Quick Gateway Candidates
                        Text(
                            text = stringResource(R.string.quick_gateways),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .horizontalScroll(rememberScrollState())
                                .padding(vertical = 6.dp),
                            horizontalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            candidates.forEach { c ->
                                val isSelected = repository.baseUrl == c
                                AssistChip(
                                    onClick = {
                                        routerUrl = c
                                        repository.baseUrl = c
                                        coroutineScope.launch {
                                            repository.refreshAll()
                                            repository.fetchSettings()
                                        }
                                    },
                                    label = {
                                        Text(
                                            c.replace("http://", "").replace(":8088", ""),
                                            color = if (isSelected) SuccessGreen else MaterialTheme.colorScheme.onSurface
                                        )
                                    },
                                    colors = AssistChipDefaults.assistChipColors(
                                        containerColor = if (isSelected) SuccessGreen.copy(alpha = 0.15f) else MaterialTheme.colorScheme.surface
                                    )
                                )
                            }
                        }
                    }
                }
            }

            // Global Security Policies Section
            item {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            text = stringResource(R.string.global_policy_section),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                        Spacer(modifier = Modifier.height(14.dp))

                        // Total Switch
                        PolicyToggleRow(
                            title = stringResource(R.string.global_switch),
                            checked = globalEnabled,
                            onCheckedChange = { globalEnabled = it }
                        )

                        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.08f))

                        // SafeSearch
                        PolicyToggleRow(
                            title = stringResource(R.string.safe_search),
                            checked = safeSearch,
                            onCheckedChange = { safeSearch = it }
                        )

                        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.08f))

                        // Block DoH/DoT
                        PolicyToggleRow(
                            title = stringResource(R.string.block_doh),
                            checked = blockDoH,
                            onCheckedChange = { blockDoH = it }
                        )

                        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp), color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.08f))

                        // Isolate New Devices
                        PolicyToggleRow(
                            title = stringResource(R.string.isolate_new),
                            checked = isolateNew,
                            onCheckedChange = { isolateNew = it }
                        )

                        Spacer(modifier = Modifier.height(16.dp))

                        Button(
                            onClick = {
                                isSavingPolicy = true
                                val newSettings = GlobalSettings(
                                    enabled = globalEnabled,
                                    pinCode = repository.pinCode.ifEmpty { null },
                                    enforceSafeSearch = safeSearch,
                                    blockDoHDoT = blockDoH,
                                    isolateNewDevices = isolateNew
                                )
                                coroutineScope.launch {
                                    val ok = repository.saveSettings(newSettings).getOrDefault(false)
                                    isSavingPolicy = false
                                    saveSuccessMsg = if (ok) "全局安全策略已成功应用并下发至路由器！" else "保存失败，请检查网络连接"
                                }
                            },
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen),
                            shape = RoundedCornerShape(12.dp),
                            enabled = !isSavingPolicy
                        ) {
                            Text(
                                stringResource(R.string.btn_apply_settings),
                                fontWeight = FontWeight.Bold
                            )
                        }

                        if (saveSuccessMsg != null) {
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                text = saveSuccessMsg ?: "",
                                color = SuccessGreen,
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Medium
                            )
                        }
                    }
                }
            }

            // System Information Section
            item {
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            text = stringResource(R.string.system_info_section),
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold
                        )
                        Spacer(modifier = Modifier.height(10.dp))

                        InfoRow(label = stringResource(R.string.client_version), value = "1.0.0 (Native Android Client)")
                        InfoRow(
                            label = stringResource(R.string.dpi_engine),
                            value = if (status?.kernelDpiReady == true) "kmod-oaf (Active)" else "Inactive"
                        )
                        InfoRow(label = stringResource(R.string.uptime), value = "${status?.uptimeSeconds ?: 0} s")
                        InfoRow(label = stringResource(R.string.detected_devices_count), value = "${status?.totalDevices ?: 0}")
                    }
                }
            }
        }
    }
}

@Composable
fun PolicyToggleRow(
    title: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Medium,
            modifier = Modifier.weight(1f)
        )
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange,
            colors = SwitchDefaults.colors(
                checkedThumbColor = SuccessGreen,
                checkedTrackColor = SuccessGreen.copy(alpha = 0.5f)
            )
        )
    }
}

@Composable
fun InfoRow(label: String, value: String) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Text(text = label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(text = value, style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
    }
}
