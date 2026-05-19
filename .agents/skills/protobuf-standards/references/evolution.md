# Protobuf Evolution & Compatibility

Rules for maintaining APIs over time without breaking downstream consumers.

## 1. Tag Number Management
- **Never Reuse Tags:** Once a tag number is assigned to a field, it is permanent. If the field is deleted, the tag must be reserved.
- **Dense Numbering:** Use tags 1 through 15 for the most frequently used fields (they take only 1 byte to encode).

## 2. Removing Fields
- **Rule:** When deleting a field, you MUST reserve both the tag number AND the field name.
- **Example:**
  ```proto
  message User {
    reserved 2, 15 to 20;
    reserved "old_email", "temporary_id";
    // ...
  }
  ```

## 3. Type Stability
- **Rule:** Never change the wire type of a field.
- **Rule:** Never change the default value of a field (causes version skew).
- **Rule:** Never use `required` (it was removed in proto3 because it breaks evolution).

## 4. Enums and Compatibility
- Adding a new value to an Enum is generally safe, provided the code handles the `UNSPECIFIED` (0) case gracefully.
- Do not reorder or delete Enum values without using `reserved`.

## 5. Persistence Pitfalls
- **No Determinism:** Protobuf serialization is NOT guaranteed to be deterministic across different languages or versions.
- **Warning:** Never use serialized Protobuf bytes as keys in databases or caches.
