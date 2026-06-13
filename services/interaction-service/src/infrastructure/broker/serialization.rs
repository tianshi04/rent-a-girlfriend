use serde::{Deserialize, Deserializer, Serializer};

pub fn serialize_timestamp<S>(
    timestamp: &Option<prost_types::Timestamp>,
    serializer: S,
) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match timestamp {
        Some(ts) => {
            let dt = chrono::DateTime::<chrono::Utc>::from_timestamp(ts.seconds, ts.nanos as u32)
                .unwrap_or_default();
            serializer.serialize_str(&dt.to_rfc3339())
        }
        None => serializer.serialize_none(),
    }
}

pub fn deserialize_timestamp<'de, D>(
    deserializer: D,
) -> Result<Option<prost_types::Timestamp>, D::Error>
where
    D: Deserializer<'de>,
{
    let s = Option::<String>::deserialize(deserializer)?;
    match s {
        Some(val) => {
            let dt =
                chrono::DateTime::parse_from_rfc3339(&val).map_err(serde::de::Error::custom)?;
            let ts = prost_types::Timestamp {
                seconds: dt.timestamp(),
                nanos: dt.timestamp_subsec_nanos() as i32,
            };
            Ok(Some(ts))
        }
        None => Ok(None),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_timestamp_serialization_format() {
        let ts = Some(prost_types::Timestamp {
            seconds: 1700000000,
            nanos: 0,
        });

        #[derive(serde::Serialize, serde::Deserialize, Debug, PartialEq)]
        #[serde(rename_all = "camelCase")]
        struct DummyEvent {
            room_id: String,
            #[serde(
                serialize_with = "serialize_timestamp",
                deserialize_with = "deserialize_timestamp"
            )]
            occurred_at: Option<prost_types::Timestamp>,
        }

        let event = DummyEvent {
            room_id: "test-room".to_string(),
            occurred_at: ts,
        };

        let serialized = serde_json::to_value(&event).unwrap();
        assert_eq!(
            serialized,
            json!({
                "roomId": "test-room",
                "occurredAt": "2023-11-14T22:13:20+00:00"
            })
        );

        let deserialized: DummyEvent = serde_json::from_value(serialized).unwrap();
        assert_eq!(deserialized, event);
    }
}
