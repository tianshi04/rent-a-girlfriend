use crate::domain::errors::DomainError;
use crate::domain::value_objects::ChatContent;
use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChatRoomStatus {
    Active,
    Locked,
}

impl ChatRoomStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            ChatRoomStatus::Active => "ACTIVE",
            ChatRoomStatus::Locked => "LOCKED",
        }
    }
}

impl std::str::FromStr for ChatRoomStatus {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "ACTIVE" => Ok(ChatRoomStatus::Active),
            "LOCKED" => Ok(ChatRoomStatus::Locked),
            _ => Err(format!("Unknown ChatRoomStatus: {}", s)),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChatRoom {
    pub room_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub status: ChatRoomStatus,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChatMessage {
    pub message_id: String,
    pub room_id: String,
    pub sender_id: String,
    pub content: String,
    pub created_at: DateTime<Utc>,
}

impl ChatRoom {
    pub fn new(
        room_id: String,
        booking_id: String,
        client_id: String,
        companion_id: String,
        status: ChatRoomStatus,
        created_at: DateTime<Utc>,
        updated_at: DateTime<Utc>,
    ) -> Self {
        Self {
            room_id,
            booking_id,
            client_id,
            companion_id,
            status,
            created_at,
            updated_at,
        }
    }

    pub fn create(booking_id: String, client_id: String, companion_id: String) -> Self {
        let now = Utc::now();
        Self {
            room_id: Uuid::new_v4().to_string(),
            booking_id,
            client_id,
            companion_id,
            status: ChatRoomStatus::Active,
            created_at: now,
            updated_at: now,
        }
    }

    pub fn lock(&mut self) -> Result<(), DomainError> {
        // [INV-I03] Once locked, cannot change back to active.
        // We only perform the update if it's currently active.
        if self.status == ChatRoomStatus::Active {
            self.status = ChatRoomStatus::Locked;
            self.updated_at = Utc::now();
        }
        Ok(())
    }

    pub fn validate_sender(&self, sender_id: &str) -> Result<(), DomainError> {
        // [INV-I01] Only participant IDs (Client, Companion) can write in this room.
        if sender_id != self.client_id && sender_id != self.companion_id {
            return Err(DomainError::UnauthorizedSender {
                room_id: self.room_id.clone(),
                user_id: sender_id.to_string(),
            });
        }
        Ok(())
    }

    pub fn validate_active(&self) -> Result<(), DomainError> {
        // [INV-I02] Cannot send message if ChatRoom is LOCKED.
        if self.status == ChatRoomStatus::Locked {
            return Err(DomainError::ChatRoomLocked(self.booking_id.clone()));
        }
        Ok(())
    }

    pub fn send_message(
        &self,
        sender_id: &str,
        content: ChatContent,
    ) -> Result<ChatMessage, DomainError> {
        self.validate_sender(sender_id)?;
        self.validate_active()?;

        Ok(ChatMessage {
            message_id: Uuid::new_v4().to_string(),
            room_id: self.room_id.clone(),
            sender_id: sender_id.to_string(),
            content: content.into_inner(),
            created_at: Utc::now(),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chat_room_creation() {
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        assert_eq!(room.booking_id, "booking-123");
        assert_eq!(room.client_id, "client-789");
        assert_eq!(room.companion_id, "companion-456");
        assert_eq!(room.status, ChatRoomStatus::Active);
    }

    #[test]
    fn test_validate_sender() {
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        // Client and Companion should be valid
        assert!(room.validate_sender("client-789").is_ok());
        assert!(room.validate_sender("companion-456").is_ok());

        // Any other user should be unauthorized
        let res = room.validate_sender("unauthorized-user");
        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::UnauthorizedSender { room_id, user_id } => {
                assert_eq!(room_id, room.room_id);
                assert_eq!(user_id, "unauthorized-user");
            }
            _ => panic!("Expected UnauthorizedSender error"),
        }
    }

    #[test]
    fn test_locking_room() {
        let mut room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        assert_eq!(room.status, ChatRoomStatus::Active);
        assert!(room.lock().is_ok());
        assert_eq!(room.status, ChatRoomStatus::Locked);

        // Lock again should remain locked and not fail
        assert!(room.lock().is_ok());
        assert_eq!(room.status, ChatRoomStatus::Locked);
    }

    #[test]
    fn test_send_message_in_locked_room_fails() {
        let mut room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        let content = ChatContent::new("Hello".to_string());

        // Message should be sent successfully when room is active
        assert!(room.send_message("client-789", content.clone()).is_ok());

        // Lock the room
        assert!(room.lock().is_ok());

        // Message sending should fail
        let res = room.send_message("client-789", content);
        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::ChatRoomLocked(booking_id) => {
                assert_eq!(booking_id, "booking-123");
            }
            _ => panic!("Expected ChatRoomLocked error"),
        }
    }
}
