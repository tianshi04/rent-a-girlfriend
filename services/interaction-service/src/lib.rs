pub mod application;
pub mod domain;
pub mod infrastructure;
pub mod interfaces;

pub mod proto {
    tonic::include_proto!("interaction.v1");
}
