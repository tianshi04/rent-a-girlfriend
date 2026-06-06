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

    // Compile tonic gRPC proto files
    tonic_prost_build::configure().compile_protos(&[proto_path], &["../../contracts"])?;
    Ok(())
}
