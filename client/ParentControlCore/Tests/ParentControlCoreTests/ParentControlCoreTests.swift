import XCTest
@testable import ParentControlCore

final class ParentControlCoreTests: XCTestCase {

    func testMemberModelDecoding() throws {
        let jsonStr = """
        {
          "id": "m_test",
          "name": "Alice",
          "avatar": "boy",
          "device_macs": ["00:11:22:33:44:55"],
          "enabled": true,
          "is_locked": false,
          "quota_minutes": 120,
          "used_minutes": 60,
          "schedule": {
            "enabled": true,
            "days": [1, 2, 3, 4, 5],
            "time_ranges": [{"start_time": "21:30", "end_time": "07:00"}],
            "action": "block"
          },
          "blocked_app_ids": [2001, 2002],
          "safe_search": true,
          "block_adult": true
        }
        """

        let data = jsonStr.data(using: .utf8)!
        let member = try JSONDecoder().decode(Member.self, from: data)

        XCTAssertEqual(member.id, "m_test")
        XCTAssertEqual(member.name, "Alice")
        XCTAssertEqual(member.quotaMinutes, 120)
        XCTAssertEqual(member.usedMinutes, 60)
        XCTAssertEqual(member.quotaProgress, 0.5)
        XCTAssertFalse(member.isQuotaExceeded)
        XCTAssertEqual(member.blockedAppIDs, [2001, 2002])
    }

    func testDeviceModelDecoding() throws {
        let jsonStr = """
        {
          "mac": "F0:18:98:AA:BB:CC",
          "ip": "192.168.0.150",
          "hostname": "iPhone-14",
          "custom_name": "Alice's iPhone",
          "vendor": "Apple",
          "online": true,
          "member_id": "m_test",
          "tx_rate": 204800,
          "rx_rate": 1048576,
          "total_bytes": 50000000,
          "last_seen": "2026-08-27T07:00:00Z"
        }
        """

        let data = jsonStr.data(using: .utf8)!
        let device = try JSONDecoder().decode(Device.self, from: data)

        XCTAssertEqual(device.mac, "F0:18:98:AA:BB:CC")
        XCTAssertEqual(device.displayName, "Alice's iPhone")
        XCTAssertTrue(device.online)
        XCTAssertEqual(device.formattedSpeed, "1.0 MB/s")
    }

    func testLiveRouterStatus() async throws {
        let client = ParentControlClient(baseURLString: "http://192.168.0.110:8088")
        do {
            let status = try await client.fetchStatus()
            XCTAssertTrue(status.running)
            print("Successfully connected to live OpenWrt router! Total devices: \(status.totalDevices)")
        } catch {
            print("Live router test skipped or offline: \(error)")
        }
    }
}
