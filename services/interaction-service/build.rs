fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Set PROTOC environment variable if our downloaded protoc exists
    let local_protoc = std::env::current_dir()
        .unwrap()
        .join("third_party")
        .join("protoc")
        .join("bin")
        .join("protoc.exe");
    if local_protoc.exists() {
        unsafe {
            std::env::set_var("PROTOC", local_protoc);
        }
    }

    // If we're building in a chef-cook/skeleton stage where contracts are not yet present,
    // skip proto compilation to avoid failing the cache build.
    let proto_path = "../../contracts/interaction/v1/service/interaction_service.proto";
    if !std::path::Path::new(proto_path).exists() {
        println!(
            "cargo:warning=Proto file not found: {}. Skipping compilation.",
            proto_path
        );
        return Ok(());
    }

    // Compile tonic gRPC proto files and events
    tonic_prost_build::configure()
        .type_attribute(".", "#[derive(serde::Serialize, serde::Deserialize)]")
        .type_attribute(".", "#[serde(rename_all = \"camelCase\")]")
        .field_attribute(
            "occurred_at",
            "#[serde(serialize_with = \"crate::infrastructure::broker::serialization::serialize_timestamp\", deserialize_with = \"crate::infrastructure::broker::serialization::deserialize_timestamp\")]"
        )
        .compile_protos(
            &[
                "../../contracts/interaction/v1/service/interaction_service.proto",
                "../../contracts/interaction/v1/events/chat_room_created.proto",
                "../../contracts/interaction/v1/events/chat_room_locked.proto",
                "../../contracts/interaction/v1/events/chat_room_creation_failed.proto",
                "../../contracts/interaction/v1/events/review_submitted.proto",
                "../../contracts/interaction/v1/events/review_hidden.proto",
            ],
            &["../../contracts"],
        )?;
    Ok(())
}
