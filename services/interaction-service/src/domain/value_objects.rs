use crate::domain::errors::DomainError;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct Rating(i32);

impl Rating {
    pub fn new(val: i32) -> Result<Self, DomainError> {
        if val < 1 || val > 5 {
            return Err(DomainError::InvalidRating(val));
        }
        Ok(Self(val))
    }

    pub fn value(&self) -> i32 {
        self.0
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChatContent(String);

impl ChatContent {
    pub fn new(content: String) -> Self {
        Self(content)
    }

    pub fn value(&self) -> &str {
        &self.0
    }

    pub fn into_inner(self) -> String {
        self.0
    }
}
