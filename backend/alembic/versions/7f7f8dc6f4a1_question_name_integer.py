"""Change question.name to integer

Revision ID: 7f7f8dc6f4a1
Revises: b63ae0f0f5b0
Create Date: 2025-11-09 13:40:00.000000
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision: str = "7f7f8dc6f4a1"
down_revision: Union[str, Sequence[str], None] = "b63ae0f0f5b0"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.alter_column(
        "question",
        "name",
        existing_type=sa.String(length=50),
        type_=sa.Integer(),
        existing_nullable=False,
    )


def downgrade() -> None:
    op.alter_column(
        "question",
        "name",
        existing_type=sa.Integer(),
        type_=sa.String(length=50),
        existing_nullable=False,
    )
