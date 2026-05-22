fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Set PROTOC environment variable if our downloaded protoc exists
    let local_protoc = std::env::current_dir()
        .unwrap()
        .join("third_party")
        .join("protoc")
        .join("bin")
        .join("protoc.exe");
    if local_protoc.exists() {
        std::env::set_var("PROTOC", local_protoc);
    }

    // Compile tonic gRPC proto files
    tonic_build::configure()
        .compile(
            &["../../contracts/interaction/v1/service/interaction_service.proto"],
            &["../../contracts"],
        )?;
    Ok(())
}
