import Foundation
import ParentControlCore

// MARK: - C/JNI Export Interface for Android Cross-Platform Interop

@_cdecl("parentcontrol_create_client")
public func parentcontrol_create_client(url: UnsafePointer<CChar>?) -> UnsafeMutableRawPointer? {
    guard let url = url else { return nil }
    let urlStr = String(cString: url)
    let client = ParentControlClient(baseURLString: urlStr)
    let unmanaged = Unmanaged.passRetained(client)
    return unmanaged.toOpaque()
}

@_cdecl("parentcontrol_destroy_client")
public func parentcontrol_destroy_client(ptr: UnsafeMutableRawPointer?) {
    guard let ptr = ptr else { return }
    Unmanaged<ParentControlClient>.fromOpaque(ptr).release()
}

@_cdecl("parentcontrol_fetch_status_json")
public func parentcontrol_fetch_status_json(
    ptr: UnsafeMutableRawPointer?,
    completion: @convention(c) (UnsafePointer<CChar>?, UnsafePointer<CChar>?) -> Void
) {
    guard let ptr = ptr else {
        completion(nil, "Null client pointer")
        return
    }
    let client = Unmanaged<ParentControlClient>.fromOpaque(ptr).takeUnretainedValue()

    Task {
        do {
            let status = try await client.fetchStatus()
            let data = try JSONEncoder().encode(status)
            if let jsonStr = String(data: data, encoding: .utf8) {
                jsonStr.withCString { cStr in
                    completion(cStr, nil)
                }
            } else {
                completion(nil, "Encoding error")
            }
        } catch {
            let errStr = error.localizedDescription
            errStr.withCString { cStr in
                completion(nil, cStr)
            }
        }
    }
}

@_cdecl("parentcontrol_lock_member")
public func parentcontrol_lock_member(
    ptr: UnsafeMutableRawPointer?,
    memberId: UnsafePointer<CChar>?,
    completion: @convention(c) (Bool, UnsafePointer<CChar>?) -> Void
) {
    guard let ptr = ptr, let memberId = memberId else {
        completion(false, "Invalid arguments")
        return
    }
    let client = Unmanaged<ParentControlClient>.fromOpaque(ptr).takeUnretainedValue()
    let idStr = String(cString: memberId)

    Task {
        do {
            try await client.lockMember(id: idStr)
            completion(true, nil)
        } catch {
            let errStr = error.localizedDescription
            errStr.withCString { cStr in
                completion(false, cStr)
            }
        }
    }
}
