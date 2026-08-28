package xyz.hamguy.parentcontrol.ui.screens

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch
import xyz.hamguy.parentcontrol.R
import xyz.hamguy.parentcontrol.repository.ParentControlRepository
import xyz.hamguy.parentcontrol.ui.theme.SuccessGreen

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    repository: ParentControlRepository,
    modifier: Modifier = Modifier
) {
    var routerUrl by remember { mutableStateOf(repository.baseUrl) }
    var pinCode by remember { mutableStateOf(repository.pinCode) }
    var saveSuccess by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    val candidates = listOf(
        "http://192.168.0.110:8088",
        "http://192.168.1.1:8088",
        "http://192.168.0.1:8088",
        "http://192.168.31.1:8088"
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
        Column(
            modifier = modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(14.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        text = "Router Gateway Connection",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Spacer(modifier = Modifier.height(12.dp))

                    OutlinedTextField(
                        value = routerUrl,
                        onValueChange = { routerUrl = it },
                        label = { Text("Router API URL") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    // Quick Gateway Candidates
                    Text(
                        text = "Quick Select:",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .horizontalScroll(rememberScrollState())
                            .padding(vertical = 4.dp),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        candidates.forEach { c ->
                            AssistChip(
                                onClick = {
                                    routerUrl = c
                                    repository.baseUrl = c
                                    coroutineScope.launch { repository.refreshAll() }
                                },
                                label = { Text(c.replace("http://", "").replace(":8088", "")) }
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    OutlinedTextField(
                        value = pinCode,
                        onValueChange = { pinCode = it },
                        label = { Text("Router PIN Code (Optional)") },
                        placeholder = { Text("Enter 4-8 digit PIN") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    Button(
                        onClick = {
                            repository.baseUrl = routerUrl
                            repository.pinCode = pinCode
                            saveSuccess = true
                            coroutineScope.launch {
                                repository.refreshAll()
                            }
                        },
                        modifier = Modifier.fillMaxWidth(),
                        colors = ButtonDefaults.buttonColors(containerColor = SuccessGreen)
                    ) {
                        Text("Save & Connect", fontWeight = FontWeight.Bold)
                    }

                    if (saveSuccess) {
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(
                            text = "Settings updated successfully!",
                            color = SuccessGreen,
                            style = MaterialTheme.typography.bodySmall
                        )
                    }
                }
            }

            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(14.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        text = "About ParentControl",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(text = "Version: 1.0.0 (Native Android Client)", style = MaterialTheme.typography.bodyMedium)
                    Text(text = "Supports JNI Swift Core bridge and direct HTTP REST communication with DPI inspection.", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}
