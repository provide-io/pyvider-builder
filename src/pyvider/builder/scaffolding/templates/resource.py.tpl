"""
{{ component_name }} resource implementation.
"""
from attrs import define, field
from pyvider.resources import register_resource
from pyvider.resources.base import BaseResource
from pyvider.resources.context import ResourceContext
from pyvider.schema import PvsSchema, a_str, s_resource
from pyvider.telemetry import logger


@define(frozen=True)
class {{ component_class_name }}Config:
    """Configuration for the {{ component_name }} resource."""
    example_attribute: str = field()


@define(frozen=True)
class {{ component_class_name }}State:
    """State for the {{ component_name }} resource."""
    id: str = field()
    example_attribute: str = field()


@register_resource("{{ provider_name }}_{{ component_name }}")
class {{ component_class_name }}Resource(
    BaseResource[
        "{{ provider_name }}_{{ component_name }}", 
        {{ component_class_name }}State, 
        {{ component_class_name }}Config
    ]
):
    """{{ component_name }} resource implementation."""
    
    config_class = {{ component_class_name }}Config
    state_class = {{ component_class_name }}State

    @classmethod
    def get_schema(cls) -> PvsSchema:
        """Define the schema for the {{ component_name }} resource."""
        return s_resource({
            "id": a_str(computed=True, description="Resource identifier"),
            "example_attribute": a_str(
                required=True, 
                description="An example attribute"
            ),
        })

    async def read(self, ctx: ResourceContext) -> {{ component_class_name }}State | None:
        """Read the current state of the resource."""
        logger.info(f"Reading resource {ctx.state.id if ctx.state else 'new'}")
        
        # TODO: Implement actual read logic
        if ctx.state:
            return ctx.state
        return None

    async def plan(
        self, ctx: ResourceContext
    ) -> tuple[{{ component_class_name }}State | None, bytes]:
        """Plan changes for the resource."""
        logger.info("Planning resource changes")
        
        # Generate ID if creating new resource
        resource_id = ctx.state.id if ctx.state else f"{{ provider_name }}-{id(ctx)}"
        
        planned_state = self.state_class(
            id=resource_id,
            example_attribute=ctx.config.example_attribute,
        )
        
        return planned_state, b""

    async def apply(
        self, ctx: ResourceContext
    ) -> tuple[{{ component_class_name }}State | None, bytes]:
        """Apply changes to create or update the resource."""
        logger.info("Applying resource changes")
        
        # TODO: Implement actual apply logic
        return ctx.planned_state, b""

    async def delete(self, ctx: ResourceContext) -> None:
        """Delete the resource."""
        logger.info(f"Deleting resource {ctx.state.id if ctx.state else 'unknown'}")
        
        # TODO: Implement actual delete logic
        pass
